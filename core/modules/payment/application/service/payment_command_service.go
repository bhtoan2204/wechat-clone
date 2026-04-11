package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/entity"
	repos "go-socket/core/modules/payment/domain/repos"
	domainservice "go-socket/core/modules/payment/domain/service"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PaymentCommandService interface {
	CreatePayment(ctx context.Context, req *in.CreatePaymentRequest) (*out.CreatePaymentResponse, error)
	ProcessWebhook(ctx context.Context, req *in.ProcessWebhookRequest) (*out.ProcessWebhookResponse, error)
}

type paymentCommandService struct {
	baseRepo         repos.Repos
	providerRegistry domainservice.PaymentProviderRegistry
}

func NewPaymentCommandService(
	baseRepo repos.Repos,
	providerRegistry domainservice.PaymentProviderRegistry,
) PaymentCommandService {
	return &paymentCommandService{
		baseRepo:         baseRepo,
		providerRegistry: providerRegistry,
	}
}

func (s *paymentCommandService) CreatePayment(
	ctx context.Context,
	req *in.CreatePaymentRequest,
) (*out.CreatePaymentResponse, error) {
	log := logging.FromContext(ctx).Named("CreatePayment")
	if err := wrapPaymentValidation(req.Validate()); err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	intent, err := entity.NewPaymentIntent(
		uuid.New().String(),
		req.Provider,
		req.Amount,
		req.Currency,
		req.DebitAccountID,
		req.CreditAccountID,
		now,
	)
	if err != nil {
		return nil, wrapPaymentValidation(err)
	}

	provider, err := s.providerRegistry.Get(intent.Provider)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	// Provider I/O must stay outside the database transaction. We persist the
	// local intent first, then persist the provider outcome with its outbox
	// side effects in a follow-up transaction.
	if err := s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return stackErr.Error(tx.ProviderPaymentRepository().CreatePaymentIntent(
			ctx,
			intent,
			intent.CreatedEvent(req.Metadata, now),
		))
	}); err != nil {
		if errors.Is(err, repos.ErrProviderPaymentDuplicateIntent) {
			return nil, fmt.Errorf("%v: %s", ErrDuplicatePayment, intent.TransactionID)
		}
		return nil, stackErr.Error(err)
	}

	creation, err := provider.CreatePayment(ctx, intent, req.Metadata)
	if err != nil {
		log.Errorw("provider create payment failed",
			"provider", provider.Name(),
			"transaction_id", intent.TransactionID,
			zap.Error(err),
		)
		if persistErr := s.markCreateFailed(ctx, intent); persistErr != nil {
			log.Errorw("failed to persist create-payment failure state",
				"provider", provider.Name(),
				"transaction_id", intent.TransactionID,
				zap.Error(persistErr),
			)
		}
		return nil, stackErr.Error(err)
	}

	if _, err := s.applyProviderOutcome(ctx, intent, creation.Result, creation.CheckoutURL, true); err != nil {
		return nil, stackErr.Error(err)
	}

	log.Infow("payment created",
		zap.String("provider", creation.Provider),
		zap.String("transaction_id", intent.TransactionID),
		zap.String("status", intent.Status),
		zap.String("external_ref", intent.ExternalRef),
	)

	return &out.CreatePaymentResponse{
		Provider:      creation.Provider,
		TransactionID: intent.TransactionID,
		ExternalRef:   intent.ExternalRef,
		Status:        intent.Status,
		CheckoutURL:   creation.CheckoutURL,
	}, nil
}

func (s *paymentCommandService) ProcessWebhook(
	ctx context.Context,
	req *in.ProcessWebhookRequest,
) (*out.ProcessWebhookResponse, error) {
	provider, err := s.providerRegistry.Get(req.Provider)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	webhook, err := provider.ParseWebhook(ctx, []byte(req.Payload), req.Signature)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	intent, err := s.findIntent(ctx, webhook.Provider, webhook.Result)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := wrapPaymentValidation(intent.ValidateProviderResult(webhook.Result.Amount, webhook.Result.Currency)); err != nil {
		return nil, stackErr.Error(err)
	}

	duplicate, err := s.applyProviderOutcome(ctx, intent, webhook.Result, "", false)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.ProcessWebhookResponse{
		Provider:      intent.Provider,
		TransactionID: intent.TransactionID,
		ExternalRef:   intent.ExternalRef,
		Status:        intent.Status,
		Duplicate:     duplicate,
		LedgerPosted:  false,
	}, nil
}

func (s *paymentCommandService) applyProviderOutcome(
	ctx context.Context,
	intent *entity.PaymentIntent,
	result entity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
) (bool, error) {
	if entity.NormalizePaymentStatus(result.Status) == entity.PaymentStatusSuccess {
		handled, err := s.finalizeSuccessfulPayment(ctx, intent, result, checkoutURL, emitCheckoutEvent)
		return handled, stackErr.Error(err)
	}

	return false, stackErr.Error(s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		updatedAt := time.Now().UTC()
		if err := intent.ApplyProviderResult(result, updatedAt); err != nil {
			return stackErr.Error(err)
		}

		outboxEvents := checkoutSessionEvents(intent, checkoutURL, updatedAt, emitCheckoutEvent)
		if intent.IsFailed() {
			outboxEvents = append(outboxEvents, intent.FailedEvent(intent.CurrentProviderResult(result), updatedAt))
		}

		return stackErr.Error(tx.ProviderPaymentRepository().SavePaymentIntent(ctx, intent, outboxEvents...))
	}))
}

func (s *paymentCommandService) finalizeSuccessfulPayment(
	ctx context.Context,
	intent *entity.PaymentIntent,
	result entity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
) (bool, error) {
	store := s.baseRepo.ProviderPaymentRepository()
	idempotencyKey := intent.PaymentIdempotencyKey(result.EventID, result.ExternalRef)
	processed, err := store.IsProcessed(ctx, intent.Provider, idempotencyKey)
	if err != nil {
		return false, stackErr.Error(err)
	}

	if processed {
		updatedAt := time.Now().UTC()
		if err := intent.ApplyProviderResult(result, updatedAt); err != nil {
			return false, stackErr.Error(err)
		}
		if err := store.SavePaymentIntent(ctx, intent, checkoutSessionEvents(intent, checkoutURL, updatedAt, emitCheckoutEvent)...); err != nil {
			return false, stackErr.Error(err)
		}
		return true, nil
	}

	if err := s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		updatedAt := time.Now().UTC()
		if err := intent.ApplyProviderResult(result, updatedAt); err != nil {
			return stackErr.Error(err)
		}

		processedEvent, err := intent.NewProcessedEvent(result, updatedAt)
		if err != nil {
			return stackErr.Error(err)
		}

		if err := tx.ProviderPaymentRepository().FinalizeSuccessfulPayment(
			ctx,
			intent,
			processedEvent,
			intent.SucceededEvent(intent.CurrentProviderResult(result), updatedAt),
			checkoutSessionEvents(intent, checkoutURL, updatedAt, emitCheckoutEvent)...,
		); err != nil {
			if errors.Is(err, repos.ErrProviderPaymentDuplicateProcessed) {
				return stackErr.Error(err)
			}
			return stackErr.Error(err)
		}

		return nil
	}); err != nil {
		if errors.Is(err, repos.ErrProviderPaymentDuplicateProcessed) {
			return true, nil
		}
		return false, stackErr.Error(err)
	}

	return false, nil
}

func (s *paymentCommandService) findIntent(
	ctx context.Context,
	provider string,
	result entity.PaymentProviderResult,
) (*entity.PaymentIntent, error) {
	store := s.baseRepo.ProviderPaymentRepository()

	if strings.TrimSpace(result.TransactionID) != "" {
		intent, err := store.GetIntentByTransactionID(ctx, result.TransactionID)
		if err == nil {
			return intent, nil
		}
		if !errors.Is(err, repos.ErrProviderPaymentNotFound) {
			return nil, stackErr.Error(err)
		}
	}

	if strings.TrimSpace(result.ExternalRef) != "" {
		intent, err := store.GetIntentByExternalRef(ctx, provider, result.ExternalRef)
		if err == nil {
			return intent, nil
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

func (s *paymentCommandService) markCreateFailed(ctx context.Context, intent *entity.PaymentIntent) error {
	if err := intent.MarkCreateFailed(time.Now().UTC()); err != nil {
		return stackErr.Error(err)
	}

	failedResult := intent.CurrentProviderResult(entity.PaymentProviderResult{Status: entity.PaymentStatusFailed})
	return stackErr.Error(s.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return stackErr.Error(tx.ProviderPaymentRepository().SavePaymentIntent(
			ctx,
			intent,
			intent.FailedEvent(failedResult, intent.UpdatedAt),
		))
	}))
}

func checkoutSessionEvents(
	intent *entity.PaymentIntent,
	checkoutURL string,
	occurredAt time.Time,
	emitCheckoutEvent bool,
) []eventpkg.Event {
	if !emitCheckoutEvent || !intent.ShouldEmitCheckoutSessionCreated(checkoutURL) {
		return nil
	}
	return []eventpkg.Event{intent.CheckoutSessionCreatedEvent(checkoutURL, occurredAt)}
}

func wrapPaymentValidation(err error) error {
	if err == nil {
		return nil
	}
	return stackErr.Error(fmt.Errorf("%v: %s", ErrValidation, err.Error()))
}
