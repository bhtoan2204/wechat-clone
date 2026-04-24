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
	"wechat-clone/core/shared/finance"
	"wechat-clone/core/shared/infra/lock"
	"wechat-clone/core/shared/pkg/actorctx"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

//go:generate mockgen -package=service -destination=payment_command_service_mock.go -source=payment_command_service.go
type PaymentCommandService interface {
	CreatePayment(ctx context.Context, req *in.CreatePaymentRequest) (*out.CreatePaymentResponse, error)
	CreateWithdrawal(ctx context.Context, req *in.CreateWithdrawalRequest) (*out.CreateWithdrawalResponse, error)
	ProcessWebhook(ctx context.Context, req *in.ProcessWebhookRequest) (*out.ProcessWebhookResponse, error)
	ProcessPendingWithdrawals(ctx context.Context) error
}

type paymentCommandService struct {
	baseRepo            repos.Repos
	locker              lock.Lock
	providerRegistry    domainservice.PaymentProviderRegistry
	feePolicy           finance.StripeFeePolicy
	withdrawalBatchSize int
}

type paymentProviderOutcome struct {
	Duplicate bool
	Events    []out.PaymentIntegrationEvent
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
		feePolicy: finance.StripeFeePolicy{
			RateBPS:    appCtx.GetConfig().LedgerConfig.Stripe.FeeRateBPS,
			FlatAmount: appCtx.GetConfig().LedgerConfig.Stripe.FeeFlatAmount,
		},
		withdrawalBatchSize: appCtx.GetConfig().LedgerConfig.Stripe.WithdrawalBatchSize,
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
	feeAmount, err := s.computeStripeFee(req.Amount)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%w: %w", ErrValidation, err))
	}

	now := time.Now().UTC()
	paymentAggregate, err := paymentaggregate.NewProviderTopUpAggregate(
		uuid.New().String(),
		req.Provider,
		req.Amount,
		feeAmount,
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
		Provider:       creation.Provider,
		Workflow:       paymentAggregate.Workflow(),
		TransactionID:  paymentAggregate.TransactionID(),
		ExternalRef:    paymentAggregate.ExternalRef(),
		Amount:         intentSnapshot.Amount,
		FeeAmount:      intentSnapshot.FeeAmount,
		ProviderAmount: intentSnapshot.ProviderAmount,
		Status:         paymentAggregate.Status(),
		CheckoutURL:    creation.CheckoutURL,
	}, nil
}

func (s *paymentCommandService) CreateWithdrawal(
	ctx context.Context,
	req *in.CreateWithdrawalRequest,
) (*out.CreateWithdrawalResponse, error) {
	debitAccountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		return nil, stackErr.Error(ErrPaymentUnauthorized)
	}

	feeAmount, err := s.computeStripeFee(req.Amount)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%w: %w", ErrValidation, err))
	}

	metadata := cloneMetadata(req.Metadata)
	if strings.TrimSpace(metadata["destination_account"]) == "" {
		return nil, stackErr.Error(fmt.Errorf("%w: metadata.destination_account is required", ErrValidation))
	}

	now := time.Now().UTC()
	paymentAggregate, err := paymentaggregate.NewProviderWithdrawalAggregate(
		uuid.New().String(),
		req.Provider,
		req.Amount,
		feeAmount,
		req.Currency,
		metadata["destination_account"],
		debitAccountID,
		metadata,
		now,
	)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%w: %w", ErrValidation, err))
	}

	if err := s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return stackErr.Error(tx.ProviderPaymentRepository().Create(ctx, paymentAggregate))
	}); err != nil {
		if errors.Is(err, repos.ErrProviderPaymentDuplicateIntent) {
			return nil, stackErr.Error(fmt.Errorf("%w: %s", ErrDuplicatePayment, paymentAggregate.TransactionID()))
		}
		return nil, stackErr.Error(err)
	}

	snapshot := paymentAggregate.Snapshot()
	return &out.CreateWithdrawalResponse{
		Provider:       snapshot.Provider,
		Workflow:       snapshot.Workflow,
		TransactionID:  snapshot.TransactionID,
		ExternalRef:    snapshot.ExternalRef,
		Amount:         snapshot.Amount,
		FeeAmount:      snapshot.FeeAmount,
		ProviderAmount: snapshot.ProviderAmount,
		Status:         snapshot.Status,
	}, nil
}

func (s *paymentCommandService) ProcessPendingWithdrawals(ctx context.Context) error {
	items, err := s.baseRepo.ProviderPaymentRepository().ListPendingWithdrawals(ctx, s.effectiveWithdrawalBatchSize())
	if err != nil {
		return stackErr.Error(err)
	}

	for _, aggregate := range items {
		if aggregate == nil || aggregate.Workflow() != entity.PaymentWorkflowWithdrawal {
			continue
		}
		if err := s.processPendingWithdrawal(ctx, aggregate.TransactionID()); err != nil {
			logging.FromContext(ctx).Warnw(
				"process pending withdrawal failed",
				"transaction_id", aggregate.TransactionID(),
				zap.Error(err),
			)
		}
	}

	return nil
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

	outcome, err := s.applyProviderOutcome(ctx, paymentAggregate, webhook.Result, "", false)
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
		Duplicate:     outcome.Duplicate,
		LedgerPosted:  false,
		Events:        outcome.Events,
	}, nil
}

func (s *paymentCommandService) processPendingWithdrawal(ctx context.Context, transactionID string) error {
	log := logging.FromContext(ctx).Named("ProcessPendingWithdrawal")
	lockKey := fmt.Sprintf("payment:%s", strings.TrimSpace(transactionID))
	lockValue := uuid.NewString()

	locked, err := s.locker.AcquireLock(ctx, lockKey, lockValue, 30*time.Second, 100*time.Millisecond, 3*time.Second)
	if err != nil {
		return stackErr.Error(err)
	}
	if !locked {
		return nil
	}
	defer func() {
		released, releaseErr := s.locker.ReleaseLock(ctx, lockKey, lockValue)
		if releaseErr != nil {
			log.Errorw("failed to release withdrawal lock", "transaction_id", transactionID, zap.Error(releaseErr))
			return
		}
		if !released {
			log.Warnw("withdrawal lock was not released", "transaction_id", transactionID)
		}
	}()

	paymentAggregate, err := s.baseRepo.ProviderPaymentRepository().GetByTransactionID(ctx, transactionID)
	if err != nil {
		return stackErr.Error(err)
	}
	if paymentAggregate == nil || paymentAggregate.Workflow() != entity.PaymentWorkflowWithdrawal || paymentAggregate.Status() != entity.PaymentStatusCreating {
		return nil
	}

	provider, err := s.providerRegistry.Get(paymentAggregate.Provider())
	if err != nil {
		return stackErr.Error(err)
	}

	intentSnapshot := paymentAggregate.Snapshot()
	creation, err := provider.CreateWithdrawal(ctx, intentSnapshot, nil)
	if err != nil {
		log.Errorw("provider create withdrawal failed", "provider", paymentAggregate.Provider(), "transaction_id", transactionID, zap.Error(err))
		if persistErr := s.markCreateFailed(ctx, paymentAggregate); persistErr != nil {
			log.Errorw("failed to persist withdrawal failure state", "provider", paymentAggregate.Provider(), "transaction_id", transactionID, zap.Error(persistErr))
		}
		return stackErr.Error(err)
	}

	if _, err := s.applyProviderOutcome(ctx, paymentAggregate, creation.Result, "", false); err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (s *paymentCommandService) applyProviderOutcome(
	ctx context.Context,
	paymentAggregate *paymentaggregate.PaymentIntentAggregate,
	result entity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
) (paymentProviderOutcome, error) {
	mutation, err := paymentAggregate.ApplyProviderOutcome(result, checkoutURL, emitCheckoutEvent, time.Now().UTC())
	if err != nil {
		return paymentProviderOutcome{}, stackErr.Error(err)
	}
	if !mutation.Persist {
		return paymentProviderOutcome{Duplicate: mutation.Duplicate}, nil
	}

	events, err := paymentIntegrationEventsFromDomainEvents(paymentAggregate.PendingOutboxEvents())
	if err != nil {
		return paymentProviderOutcome{}, stackErr.Error(err)
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
			return paymentProviderOutcome{Duplicate: true}, nil
		}
		return paymentProviderOutcome{}, stackErr.Error(persistErr)
	}

	return paymentProviderOutcome{
		Duplicate: mutation.Duplicate,
		Events:    events,
	}, nil
}

func paymentIntegrationEventsFromDomainEvents(events []eventpkg.Event) ([]out.PaymentIntegrationEvent, error) {
	if len(events) == 0 {
		return nil, nil
	}

	serializer := eventpkg.NewSerializer()
	items := make([]out.PaymentIntegrationEvent, 0, len(events))
	for _, evt := range events {
		data, err := serializer.Marshal(evt.EventData)
		if err != nil {
			return nil, stackErr.Error(fmt.Errorf("marshal payment integration event failed: %w", err))
		}
		items = append(items, out.PaymentIntegrationEvent{
			Name:     evt.EventName,
			DataJson: string(data),
		})
	}
	return items, nil
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
	if err := s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return stackErr.Error(tx.ProviderPaymentRepository().Save(ctx, paymentAggregate))
	}); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (s *paymentCommandService) computeStripeFee(amount int64) (int64, error) {
	return s.feePolicy.Compute(amount)
}

func (s *paymentCommandService) effectiveWithdrawalBatchSize() int {
	if s.withdrawalBatchSize <= 0 {
		return 20
	}
	return s.withdrawalBatchSize
}

func (s *paymentCommandService) resolveCreatePaymentCreditAccount(
	ctx context.Context,
	req *in.CreatePaymentRequest,
) (string, error) {
	_ = req
	actorAccountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		return "", stackErr.Error(ErrPaymentUnauthorized)
	}

	return actorAccountID, nil
}

func isPaymentValidationError(err error) bool {
	return errors.Is(err, entity.ErrPaymentWorkflowInvalid) ||
		errors.Is(err, entity.ErrPaymentTransactionIDRequired) ||
		errors.Is(err, entity.ErrPaymentAmountInvalid) ||
		errors.Is(err, entity.ErrPaymentFeeAmountInvalid) ||
		errors.Is(err, entity.ErrPaymentProviderAmountInvalid) ||
		errors.Is(err, entity.ErrPaymentCurrencyRequired) ||
		errors.Is(err, entity.ErrPaymentCreditAccountRequired) ||
		errors.Is(err, entity.ErrPaymentStatusInvalid) ||
		errors.Is(err, entity.ErrPaymentProviderAmountMismatch) ||
		errors.Is(err, entity.ErrPaymentProviderCurrencyMismatch) ||
		errors.Is(err, entity.ErrPaymentDestinationAccountRequired) ||
		errors.Is(err, entity.ErrPaymentDebitAccountRequired) ||
		errors.Is(err, entity.ErrPaymentAccountsConflict) ||
		errors.Is(err, paymentaggregate.ErrPaymentIntentOccurredAtRequired)
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = strings.TrimSpace(value)
	}

	return cloned
}
