package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	paymentin "go-socket/core/modules/payment/application/dto/in"
	paymentout "go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/entity"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/providers"
	sharedevents "go-socket/core/shared/contracts/events"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

const paymentAggregateType = "payment"

type PaymentService struct {
	intentStore      PaymentIntentStore
	providerRegistry *providers.ProviderRegistry
}

func NewPaymentService(intentStore PaymentIntentStore, providerRegistry *providers.ProviderRegistry) *PaymentService {
	return &PaymentService{
		intentStore:      intentStore,
		providerRegistry: providerRegistry,
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req *paymentin.CreatePaymentRequest) (*paymentout.CreatePaymentResponse, error) {
	log := logging.FromContext(ctx).Named("CreatePayment")
	req.Normalize()
	if err := wrapValidation(req.Validate()); err != nil {
		return nil, stackErr.Error(err)
	}

	provider, err := s.providerRegistry.Get(req.Provider)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	intent, err := entity.NewPaymentIntent(
		req.TransactionID,
		req.Provider,
		req.Amount,
		req.Currency,
		req.DebitAccountID,
		req.CreditAccountID,
		now,
	)
	if err != nil {
		return nil, wrapValidation(err)
	}

	if err := s.intentStore.WithTransaction(ctx, func(store PaymentIntentStore) error {
		if err := store.CreateIntent(ctx, intent); err != nil {
			return err
		}
		return store.AppendOutboxEvent(ctx, newPaymentCreatedEvent(intent, req.Metadata))
	}); err != nil {
		if errors.Is(err, paymentrepos.ErrProviderPaymentDuplicateIntent) {
			return nil, fmt.Errorf("%v: %s", ErrDuplicatePayment, req.TransactionID)
		}
		return nil, stackErr.Error(err)
	}

	response, err := provider.CreatePayment(ctx, providers.CreatePaymentRequest{
		TransactionID:   req.TransactionID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		DebitAccountID:  req.DebitAccountID,
		CreditAccountID: req.CreditAccountID,
		Metadata:        req.Metadata,
	})
	if err != nil {
		logging.FromContext(ctx).Errorw("provider create payment failed",
			"provider", provider.Name(),
			"transaction_id", req.TransactionID,
			zap.Error(err),
		)
		if stateErr := intent.SetProviderState("", entity.PaymentStatusFailed, time.Now().UTC()); stateErr == nil {
			_ = s.updateIntentStatus(ctx, intent.TransactionID, intent.Status)
		}
		return nil, stackErr.Error(err)
	}

	targetStatus := entity.NormalizePaymentStatusOrPending(response.Status)

	persistedIntent, err := s.intentStore.GetIntentByTransactionID(ctx, req.TransactionID)
	if err != nil {
		if errors.Is(err, paymentrepos.ErrProviderPaymentNotFound) {
			return nil, fmt.Errorf("%v: %s", ErrPaymentIntentNotFound, req.TransactionID)
		}
		return nil, stackErr.Error(err)
	}
	if err := s.intentStore.WithTransaction(ctx, func(store PaymentIntentStore) error {
		if err := persistedIntent.SetProviderState(response.ExternalRef, targetStatus, time.Now().UTC()); err != nil {
			return err
		}
		if err := store.UpdateIntentProviderState(ctx, persistedIntent.TransactionID, persistedIntent.ExternalRef, persistedIntent.Status); err != nil {
			return err
		}

		if response.CheckoutURL != "" || persistedIntent.ExternalRef != "" {
			if err := store.AppendOutboxEvent(ctx, newPaymentCheckoutSessionCreatedEvent(persistedIntent, response, persistedIntent.Status)); err != nil {
				return stackErr.Error(err)
			}
		}

		if persistedIntent.Status == entity.PaymentStatusSuccess {
			return s.finalizeSuccessfulPaymentTx(ctx, store, persistedIntent, &providers.PaymentResult{
				TransactionID: response.TransactionID,
				Status:        persistedIntent.Status,
				Amount:        persistedIntent.Amount,
				Currency:      persistedIntent.Currency,
				ExternalRef:   persistedIntent.ExternalRef,
			})
		}
		if persistedIntent.Status == entity.PaymentStatusFailed {
			return store.AppendOutboxEvent(ctx, newPaymentFailedEvent(persistedIntent, &providers.PaymentResult{
				TransactionID: response.TransactionID,
				Status:        persistedIntent.Status,
				Amount:        persistedIntent.Amount,
				Currency:      persistedIntent.Currency,
				ExternalRef:   persistedIntent.ExternalRef,
			}))
		}
		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	log.Infow("payment created",
		zap.String("provider", provider.Name()),
		zap.String("transaction_id", response.TransactionID),
		zap.String("status", targetStatus),
		zap.String("external_ref", response.ExternalRef),
	)

	return &paymentout.CreatePaymentResponse{
		Provider:      strings.ToLower(provider.Name()),
		TransactionID: response.TransactionID,
		ExternalRef:   response.ExternalRef,
		Status:        targetStatus,
		CheckoutURL:   response.CheckoutURL,
	}, nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, providerName string, payload []byte, signature string) (*paymentout.ProcessWebhookResponse, error) {
	provider, err := s.providerRegistry.Get(providerName)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	event, err := provider.VerifyWebhook(ctx, payload, signature)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	result, err := provider.ParseEvent(ctx, event)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result.Status = entity.NormalizePaymentStatusOrPending(result.Status)

	intent, err := s.findIntent(ctx, strings.ToLower(provider.Name()), result)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := wrapValidation(intent.ValidateProviderResult(result.Amount, result.Currency)); err != nil {
		return nil, stackErr.Error(err)
	}

	if result.Status != entity.PaymentStatusSuccess {
		if err := s.intentStore.WithTransaction(ctx, func(store PaymentIntentStore) error {
			if err := intent.SetProviderState(result.ExternalRef, result.Status, time.Now().UTC()); err != nil {
				return stackErr.Error(err)
			}
			if err := store.UpdateIntentProviderState(ctx, intent.TransactionID, intent.ExternalRef, intent.Status); err != nil {
				return stackErr.Error(err)
			}
			if intent.Status == entity.PaymentStatusFailed {
				return store.AppendOutboxEvent(ctx, newPaymentFailedEvent(intent, result))
			}
			return nil
		}); err != nil {
			return nil, stackErr.Error(err)
		}
		return &paymentout.ProcessWebhookResponse{
			Provider:      intent.Provider,
			TransactionID: intent.TransactionID,
			ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
			Status:        result.Status,
		}, nil
	}

	duplicate, err := s.finalizeSuccessfulPayment(ctx, intent, result)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &paymentout.ProcessWebhookResponse{
		Provider:      intent.Provider,
		TransactionID: intent.TransactionID,
		ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
		Status:        entity.PaymentStatusSuccess,
		Duplicate:     duplicate,
		LedgerPosted:  false,
	}, nil
}

func (s *PaymentService) finalizeSuccessfulPayment(ctx context.Context, intent *entity.PaymentIntent, result *providers.PaymentResult) (bool, error) {
	idempotencyKey := intent.PaymentIdempotencyKey(result.EventID, result.ExternalRef)
	processed, err := s.intentStore.IsProcessed(ctx, intent.Provider, idempotencyKey)
	if err != nil {
		return false, err
	}
	if processed {
		if err := intent.SetProviderState(result.ExternalRef, entity.PaymentStatusSuccess, time.Now().UTC()); err != nil {
			return false, err
		}
		if err := s.intentStore.UpdateIntentProviderState(ctx, intent.TransactionID, intent.ExternalRef, intent.Status); err != nil {
			return false, err
		}
		return true, nil
	}

	if err := s.intentStore.WithTransaction(ctx, func(store PaymentIntentStore) error {
		return s.finalizeSuccessfulPaymentTx(ctx, store, intent, result)
	}); err != nil {
		if errors.Is(err, paymentrepos.ErrProviderPaymentDuplicateProcessed) {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

func (s *PaymentService) finalizeSuccessfulPaymentTx(ctx context.Context, store PaymentIntentStore, intent *entity.PaymentIntent, result *providers.PaymentResult) error {
	idempotencyKey := intent.PaymentIdempotencyKey(result.EventID, result.ExternalRef)
	processedEvent, err := entity.NewProcessedPaymentEvent(intent.Provider, idempotencyKey, intent.TransactionID, time.Now().UTC())
	if err != nil {
		return stackErr.Error(err)
	}

	if err := store.MarkProcessed(ctx, processedEvent); err != nil {
		if errors.Is(err, paymentrepos.ErrProviderPaymentDuplicateProcessed) {
			return err
		}
		return stackErr.Error(err)
	}

	if err := intent.SetProviderState(result.ExternalRef, entity.PaymentStatusSuccess, time.Now().UTC()); err != nil {
		return stackErr.Error(err)
	}
	if err := store.UpdateIntentProviderState(ctx, intent.TransactionID, intent.ExternalRef, intent.Status); err != nil {
		return stackErr.Error(err)
	}

	return store.AppendOutboxEvent(ctx, newPaymentSucceededEvent(intent, result))
}

func (s *PaymentService) findIntent(ctx context.Context, provider string, result *providers.PaymentResult) (*entity.PaymentIntent, error) {
	if strings.TrimSpace(result.TransactionID) != "" {
		intent, err := s.intentStore.GetIntentByTransactionID(ctx, result.TransactionID)
		if err == nil {
			return intent, nil
		}
		if !errors.Is(err, paymentrepos.ErrProviderPaymentNotFound) {
			return nil, stackErr.Error(err)
		}
	}

	if strings.TrimSpace(result.ExternalRef) != "" {
		intent, err := s.intentStore.GetIntentByExternalRef(ctx, provider, result.ExternalRef)
		if err == nil {
			return intent, nil
		}
		if !errors.Is(err, paymentrepos.ErrProviderPaymentNotFound) {
			return nil, stackErr.Error(err)
		}
	}

	return nil, fmt.Errorf("%v: transaction_id=%s external_ref=%s", ErrPaymentIntentNotFound, result.TransactionID, result.ExternalRef)
}

func (s *PaymentService) updateIntentStatus(ctx context.Context, transactionID, status string) error {
	return s.intentStore.UpdateIntentStatus(ctx, transactionID, status)
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func wrapValidation(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%v: %s", ErrValidation, err.Error())
}

func newPaymentCreatedEvent(intent *entity.PaymentIntent, metadata map[string]string) eventpkg.Event {
	return eventpkg.Event{
		AggregateID:   intent.TransactionID,
		AggregateType: paymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentCreated,
		EventData: sharedevents.PaymentCreatedEvent{
			PaymentID:       intent.TransactionID,
			TransactionID:   intent.TransactionID,
			Provider:        intent.Provider,
			Amount:          intent.Amount,
			Currency:        intent.Currency,
			DebitAccountID:  intent.DebitAccountID,
			CreditAccountID: intent.CreditAccountID,
			Status:          intent.Status,
			Metadata:        metadata,
			CreatedAt:       intent.CreatedAt,
		},
		CreatedAt: time.Now().Unix(),
	}
}

func newPaymentCheckoutSessionCreatedEvent(intent *entity.PaymentIntent, response *providers.CreatePaymentResponse, status string) eventpkg.Event {
	return eventpkg.Event{
		AggregateID:   intent.TransactionID,
		AggregateType: paymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentCheckoutSessionCreated,
		EventData: sharedevents.PaymentCheckoutSessionCreatedEvent{
			PaymentID:          intent.TransactionID,
			TransactionID:      intent.TransactionID,
			Provider:           intent.Provider,
			ProviderPaymentRef: response.ExternalRef,
			CheckoutURL:        response.CheckoutURL,
			Amount:             intent.Amount,
			Currency:           intent.Currency,
			Status:             status,
			OccurredAt:         time.Now().UTC(),
		},
		CreatedAt: time.Now().Unix(),
	}
}

func newPaymentSucceededEvent(intent *entity.PaymentIntent, result *providers.PaymentResult) eventpkg.Event {
	return eventpkg.Event{
		AggregateID:   intent.TransactionID,
		AggregateType: paymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentSucceeded,
		EventData: sharedevents.PaymentSucceededEvent{
			PaymentID:          intent.TransactionID,
			TransactionID:      intent.TransactionID,
			Provider:           intent.Provider,
			ProviderEventID:    result.EventID,
			ProviderEventType:  result.EventType,
			ProviderPaymentRef: coalesce(result.ExternalRef, intent.ExternalRef),
			Amount:             intent.Amount,
			Currency:           intent.Currency,
			DebitAccountID:     intent.DebitAccountID,
			CreditAccountID:    intent.CreditAccountID,
			IdempotencyKey:     fmt.Sprintf("%s:%s", sharedevents.EventPaymentSucceeded, intent.TransactionID),
			SucceededAt:        time.Now().UTC(),
		},
		CreatedAt: time.Now().Unix(),
	}
}

func newPaymentFailedEvent(intent *entity.PaymentIntent, result *providers.PaymentResult) eventpkg.Event {
	return eventpkg.Event{
		AggregateID:   intent.TransactionID,
		AggregateType: paymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentFailed,
		EventData: sharedevents.PaymentFailedEvent{
			PaymentID:          intent.TransactionID,
			TransactionID:      intent.TransactionID,
			Provider:           intent.Provider,
			ProviderEventID:    result.EventID,
			ProviderEventType:  result.EventType,
			ProviderPaymentRef: coalesce(result.ExternalRef, intent.ExternalRef),
			Amount:             intent.Amount,
			Currency:           intent.Currency,
			Status:             result.Status,
			OccurredAt:         time.Now().UTC(),
		},
		CreatedAt: time.Now().Unix(),
	}
}
