package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/payment/application/dto/in"
	"wechat-clone/core/modules/payment/application/dto/out"
	paymentaggregate "wechat-clone/core/modules/payment/domain/aggregate"
	"wechat-clone/core/modules/payment/domain/entity"
	repos "wechat-clone/core/modules/payment/domain/repos"
	domainservice "wechat-clone/core/modules/payment/domain/service"
	"wechat-clone/core/shared/infra/lock"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

//go:generate mockgen -package=service -destination=payment_command_service_mock.go -source=payment_command_service.go
type PaymentCommandService interface {
	CreatePayment(ctx context.Context, req *in.CreatePaymentRequest) (*out.CreatePaymentResponse, error)
	ProcessWebhook(ctx context.Context, req *in.ProcessWebhookRequest) (*out.ProcessWebhookResponse, error)
}

type paymentCommandService struct {
	baseRepo         repos.Repos
	locker           lock.Lock
	providerRegistry domainservice.PaymentProviderRegistry
}

func NewPaymentCommandService(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	providerRegistry domainservice.PaymentProviderRegistry,
) PaymentCommandService {
	return &paymentCommandService{
		baseRepo:         baseRepo,
		providerRegistry: providerRegistry,
		locker:           appCtx.Locker(),
	}
}

func (s *paymentCommandService) CreatePayment(
	ctx context.Context,
	req *in.CreatePaymentRequest,
) (*out.CreatePaymentResponse, error) {
	log := logging.FromContext(ctx).Named("CreatePayment")
	creditAccountID, err := s.resolveCreatePaymentCreditAccount(ctx, req)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	paymentAggregate, err := paymentaggregate.NewProviderTopUpAggregate(
		uuid.New().String(),
		req.Provider,
		req.Amount,
		req.Currency,
		creditAccountID,
		req.Metadata,
		now,
	)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%w: %w", ErrValidation, err))
	}

	provider, err := s.providerRegistry.Get(paymentAggregate.Provider())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return stackErr.Error(tx.ProviderPaymentRepository().Create(ctx, paymentAggregate))
	}); err != nil {
		if errors.Is(err, repos.ErrProviderPaymentDuplicateIntent) {
			return nil, stackErr.Error(fmt.Errorf("%w: %s", ErrDuplicatePayment, paymentAggregate.TransactionID()))
		}
		return nil, stackErr.Error(err)
	}

	intentSnapshot := paymentAggregate.Snapshot()
	creation, err := provider.CreatePayment(ctx, intentSnapshot, req.Metadata)
	if err != nil {
		log.Errorw("provider create payment failed", "provider", provider.Name(), "transaction_id", paymentAggregate.TransactionID(), zap.Error(err))
		if persistErr := s.markCreateFailed(ctx, paymentAggregate); persistErr != nil {
			log.Errorw("failed to persist create-payment failure state", "provider", provider.Name(), "transaction_id", paymentAggregate.TransactionID(), zap.Error(persistErr))
		}
		return nil, stackErr.Error(err)
	}

	if _, err := s.applyProviderOutcome(ctx, paymentAggregate, creation.Result, creation.CheckoutURL, true); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.CreatePaymentResponse{
		Provider:      creation.Provider,
		TransactionID: paymentAggregate.TransactionID(),
		ExternalRef:   paymentAggregate.ExternalRef(),
		Status:        paymentAggregate.Status(),
		CheckoutURL:   creation.CheckoutURL,
	}, nil
}

func (s *paymentCommandService) ProcessWebhook(
	ctx context.Context,
	req *in.ProcessWebhookRequest,
) (*out.ProcessWebhookResponse, error) {
	log := logging.FromContext(ctx).Named("ProcessWebhook")
	provider, err := s.providerRegistry.Get(req.Provider)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	webhook, err := provider.ParseWebhook(ctx, []byte(req.Payload), req.Signature)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if webhook.Ignored {
		return &out.ProcessWebhookResponse{
			Provider:     webhook.Provider,
			LedgerPosted: false,
		}, nil
	}

	paymentAggregate, err := s.findPaymentAggregate(ctx, webhook.Provider, webhook.Result)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	lockKey := fmt.Sprintf("payment:%s", paymentAggregate.TransactionID())
	lockValue := uuid.NewString()
	locked, err := s.locker.AcquireLock(ctx, lockKey, lockValue, 30*time.Second, 100*time.Millisecond, 3*time.Second)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if !locked {
		return nil, stackErr.Error(fmt.Errorf("acquire payment lock failed: transaction_id=%s", paymentAggregate.TransactionID()))
	}
	defer func() {
		released, err := s.locker.ReleaseLock(ctx, lockKey, lockValue)
		if err != nil {
			log.Errorw("failed to release payment lock", zap.String("transaction_id", paymentAggregate.TransactionID()), zap.String("lock_key", lockKey), zap.Error(err))
			return
		}
		if !released {
			log.Warnw("payment lock was not released", zap.String("transaction_id", paymentAggregate.TransactionID()), zap.String("lock_key", lockKey))
		}
	}()

	paymentAggregate, err = s.baseRepo.ProviderPaymentRepository().GetByTransactionID(ctx, paymentAggregate.TransactionID())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	duplicate, err := s.applyProviderOutcome(ctx, paymentAggregate, webhook.Result, "", false)
	if err != nil {
		if isPaymentValidationError(err) {
			return nil, stackErr.Error(fmt.Errorf("%w: %w", ErrValidation, err))
		}
		return nil, stackErr.Error(err)
	}

	return &out.ProcessWebhookResponse{
		Provider:      paymentAggregate.Provider(),
		TransactionID: paymentAggregate.TransactionID(),
		ExternalRef:   paymentAggregate.ExternalRef(),
		Status:        paymentAggregate.Status(),
		Duplicate:     duplicate,
		LedgerPosted:  false,
	}, nil
}

func (s *paymentCommandService) applyProviderOutcome(
	ctx context.Context,
	paymentAggregate *paymentaggregate.PaymentIntentAggregate,
	result entity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
) (bool, error) {
	mutation, err := paymentAggregate.ApplyProviderOutcome(result, checkoutURL, emitCheckoutEvent, time.Now().UTC())
	if err != nil {
		return false, stackErr.Error(err)
	}
	if !mutation.Persist {
		return mutation.Duplicate, nil
	}

	persistErr := s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		if err := tx.ProviderPaymentRepository().Save(ctx, paymentAggregate); err != nil {
			if errors.Is(err, repos.ErrProviderPaymentDuplicateProcessed) {
				return stackErr.Error(err)
			}
			return stackErr.Error(err)
		}
		return nil
	})
	if persistErr != nil {
		if errors.Is(persistErr, repos.ErrProviderPaymentDuplicateProcessed) {
			return true, nil
		}
		return false, stackErr.Error(persistErr)
	}

	return mutation.Duplicate, nil
}

func (s *paymentCommandService) findPaymentAggregate(
	ctx context.Context,
	provider string,
	result entity.PaymentProviderResult,
) (*paymentaggregate.PaymentIntentAggregate, error) {
	store := s.baseRepo.ProviderPaymentRepository()

	if strings.TrimSpace(result.TransactionID) != "" {
		paymentAggregate, err := store.GetByTransactionID(ctx, result.TransactionID)
		if err == nil {
			return paymentAggregate, nil
		}
		if !errors.Is(err, repos.ErrProviderPaymentNotFound) {
			return nil, stackErr.Error(err)
		}
	}

	if strings.TrimSpace(result.ExternalRef) != "" {
		paymentAggregate, err := store.GetByExternalRef(ctx, provider, result.ExternalRef)
		if err == nil {
			return paymentAggregate, nil
		}
		if !errors.Is(err, repos.ErrProviderPaymentNotFound) {
			return nil, stackErr.Error(err)
		}
	}

	return nil, stackErr.Error(fmt.Errorf(
		"%v: transaction_id=%s external_ref=%s",
		ErrPaymentIntentNotFound,
		result.TransactionID,
		result.ExternalRef,
	))
}

func (s *paymentCommandService) markCreateFailed(ctx context.Context, paymentAggregate *paymentaggregate.PaymentIntentAggregate) error {
	mutation, err := paymentAggregate.MarkCreateFailed(time.Now().UTC())
	if err != nil {
		return stackErr.Error(err)
	}
	if !mutation.Persist {
		return nil
	}

	return stackErr.Error(s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return stackErr.Error(tx.ProviderPaymentRepository().Save(ctx, paymentAggregate))
	}))
}

func (s *paymentCommandService) resolveCreatePaymentCreditAccount(
	ctx context.Context,
	req *in.CreatePaymentRequest,
) (string, error) {
	actorAccountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		return "", stackErr.Error(ErrPaymentUnauthorized)
	}

	requestedCreditAccountID := strings.TrimSpace(req.CreditAccountID)
	if requestedCreditAccountID != "" {
		return "", stackErr.Error(fmt.Errorf("%w: credit_account_id is server-owned and must not be provided", ErrValidation))
	}

	return actorAccountID, nil
}

func isPaymentValidationError(err error) bool {
	return errors.Is(err, entity.ErrPaymentTransactionIDRequired) ||
		errors.Is(err, entity.ErrPaymentAmountInvalid) ||
		errors.Is(err, entity.ErrPaymentCurrencyRequired) ||
		errors.Is(err, entity.ErrPaymentStatusInvalid) ||
		errors.Is(err, entity.ErrPaymentProviderAmountMismatch) ||
		errors.Is(err, entity.ErrPaymentProviderCurrencyMismatch) ||
		errors.Is(err, paymentaggregate.ErrPaymentIntentOccurredAtRequired)
}
