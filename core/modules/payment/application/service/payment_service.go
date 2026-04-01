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
	"go-socket/core/shared/pkg/logging"
)

type PaymentService struct {
	intentStore      PaymentIntentStore
	ledgerGateway    LedgerGateway
	providerRegistry *providers.ProviderRegistry
}

func NewPaymentService(intentStore PaymentIntentStore, ledgerGateway LedgerGateway, providerRegistry *providers.ProviderRegistry) *PaymentService {
	return &PaymentService{
		intentStore:      intentStore,
		ledgerGateway:    ledgerGateway,
		providerRegistry: providerRegistry,
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req *paymentin.CreatePaymentRequest) (*paymentout.CreatePaymentResponse, error) {
	req.Normalize()
	if err := wrapValidation(req.Validate()); err != nil {
		return nil, err
	}

	provider, err := s.providerRegistry.Get(req.Provider)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	intent := &entity.PaymentIntent{
		TransactionID:   req.TransactionID,
		Provider:        req.Provider,
		Amount:          req.Amount,
		Currency:        req.Currency,
		DebitAccountID:  req.DebitAccountID,
		CreditAccountID: req.CreditAccountID,
		Status:          entity.PaymentStatusCreating,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.intentStore.CreateIntent(ctx, intent); err != nil {
		if errors.Is(err, paymentrepos.ErrProviderPaymentDuplicateIntent) {
			return nil, fmt.Errorf("%v: %s", ErrDuplicatePayment, req.TransactionID)
		}
		return nil, err
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
			"error", err,
		)
		_ = s.updateIntentStatus(ctx, req.TransactionID, entity.PaymentStatusFailed)
		return nil, err
	}

	targetStatus := normalizePaymentStatus(response.Status)
	if targetStatus == "" {
		targetStatus = entity.PaymentStatusPending
	}

	persistedIntent, err := s.intentStore.GetIntentByTransactionID(ctx, req.TransactionID)
	if err != nil {
		if errors.Is(err, paymentrepos.ErrProviderPaymentNotFound) {
			return nil, fmt.Errorf("%v: %s", ErrPaymentIntentNotFound, req.TransactionID)
		}
		return nil, err
	}
	if err := s.intentStore.UpdateIntentProviderState(ctx, persistedIntent.TransactionID, response.ExternalRef, targetStatus); err != nil {
		return nil, err
	}
	if targetStatus == entity.PaymentStatusSuccess {
		_, _, err := s.finalizeSuccessfulPayment(ctx, persistedIntent, &providers.PaymentResult{
			TransactionID: response.TransactionID,
			Status:        targetStatus,
			Amount:        persistedIntent.Amount,
			Currency:      persistedIntent.Currency,
			ExternalRef:   response.ExternalRef,
		})
		if err != nil {
			return nil, err
		}
	}

	logging.FromContext(ctx).Infow("payment created",
		"provider", provider.Name(),
		"transaction_id", response.TransactionID,
		"status", targetStatus,
		"external_ref", response.ExternalRef,
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
		return nil, err
	}

	event, err := provider.VerifyWebhook(ctx, payload, signature)
	if err != nil {
		return nil, err
	}

	result, err := provider.ParseEvent(ctx, event)
	if err != nil {
		return nil, err
	}
	result.Status = normalizePaymentStatus(result.Status)

	intent, err := s.findIntent(ctx, strings.ToLower(provider.Name()), result)
	if err != nil {
		return nil, err
	}
	if err := validateResultAgainstIntent(intent, result); err != nil {
		return nil, err
	}

	if result.Status != entity.PaymentStatusSuccess {
		if err := s.intentStore.UpdateIntentProviderState(ctx, intent.TransactionID, result.ExternalRef, result.Status); err != nil {
			return nil, err
		}
		return &paymentout.ProcessWebhookResponse{
			Provider:      intent.Provider,
			TransactionID: intent.TransactionID,
			ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
			Status:        result.Status,
		}, nil
	}

	ledgerPosted, duplicate, err := s.finalizeSuccessfulPayment(ctx, intent, result)
	if err != nil {
		return nil, err
	}

	return &paymentout.ProcessWebhookResponse{
		Provider:      intent.Provider,
		TransactionID: intent.TransactionID,
		ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
		Status:        entity.PaymentStatusSuccess,
		Duplicate:     duplicate,
		LedgerPosted:  ledgerPosted,
	}, nil
}

func (s *PaymentService) finalizeSuccessfulPayment(ctx context.Context, intent *entity.PaymentIntent, result *providers.PaymentResult) (bool, bool, error) {
	idempotencyKey := paymentIdempotencyKey(intent, result)
	processed, err := s.intentStore.IsProcessed(ctx, intent.Provider, idempotencyKey)
	if err != nil {
		return false, false, err
	}
	if processed {
		if err := s.intentStore.UpdateIntentProviderState(ctx, intent.TransactionID, result.ExternalRef, entity.PaymentStatusSuccess); err != nil {
			return false, false, err
		}
		return false, true, nil
	}

	err = s.ledgerGateway.PostTransaction(ctx, LedgerPostingRequest{
		TransactionID: intent.TransactionID,
		Entries: []LedgerPostingEntry{
			{AccountID: intent.DebitAccountID, Amount: -intent.Amount},
			{AccountID: intent.CreditAccountID, Amount: intent.Amount},
		},
	})
	ledgerPosted := err == nil
	duplicate := false
	if err != nil {
		if errors.Is(err, ErrDuplicateTransaction) {
			duplicate = true
		} else {
			return false, false, err
		}
	}

	if err := s.intentStore.MarkProcessed(ctx, &entity.ProcessedPaymentEvent{
		Provider:       intent.Provider,
		IdempotencyKey: idempotencyKey,
		TransactionID:  intent.TransactionID,
		CreatedAt:      time.Now().UTC(),
	}); err != nil && !errors.Is(err, paymentrepos.ErrProviderPaymentDuplicateProcessed) {
		return false, false, err
	}

	if err := s.intentStore.UpdateIntentProviderState(ctx, intent.TransactionID, result.ExternalRef, entity.PaymentStatusSuccess); err != nil {
		return false, false, err
	}

	return ledgerPosted, duplicate, nil
}

func (s *PaymentService) findIntent(ctx context.Context, provider string, result *providers.PaymentResult) (*entity.PaymentIntent, error) {
	if strings.TrimSpace(result.TransactionID) != "" {
		intent, err := s.intentStore.GetIntentByTransactionID(ctx, result.TransactionID)
		if err == nil {
			return intent, nil
		}
		if !errors.Is(err, paymentrepos.ErrProviderPaymentNotFound) {
			return nil, err
		}
	}

	if strings.TrimSpace(result.ExternalRef) != "" {
		intent, err := s.intentStore.GetIntentByExternalRef(ctx, provider, result.ExternalRef)
		if err == nil {
			return intent, nil
		}
		if !errors.Is(err, paymentrepos.ErrProviderPaymentNotFound) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("%v: transaction_id=%s external_ref=%s", ErrPaymentIntentNotFound, result.TransactionID, result.ExternalRef)
}

func (s *PaymentService) updateIntentStatus(ctx context.Context, transactionID, status string) error {
	return s.intentStore.UpdateIntentStatus(ctx, transactionID, status)
}

func validateResultAgainstIntent(intent *entity.PaymentIntent, result *providers.PaymentResult) error {
	if result.Amount != 0 && result.Amount != intent.Amount {
		return fmt.Errorf("%v: provider amount does not match reserved payment", ErrValidation)
	}
	if currency := strings.TrimSpace(result.Currency); currency != "" && !strings.EqualFold(currency, intent.Currency) {
		return fmt.Errorf("%v: provider currency does not match reserved payment", ErrValidation)
	}
	return nil
}

func normalizePaymentStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case entity.PaymentStatusSuccess:
		return entity.PaymentStatusSuccess
	case entity.PaymentStatusFailed:
		return entity.PaymentStatusFailed
	case entity.PaymentStatusCreating:
		return entity.PaymentStatusCreating
	case entity.PaymentStatusPending:
		return entity.PaymentStatusPending
	default:
		return entity.PaymentStatusPending
	}
}

func paymentIdempotencyKey(intent *entity.PaymentIntent, result *providers.PaymentResult) string {
	if strings.TrimSpace(result.ExternalRef) != "" {
		return result.ExternalRef
	}
	if strings.TrimSpace(intent.ExternalRef) != "" {
		return intent.ExternalRef
	}
	return intent.TransactionID
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
