package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	ledgerrepo "go-socket/core/modules/ledger/infra/persistent/repository"
	"go-socket/core/modules/ledger/providers"
	"go-socket/core/shared/pkg/logging"
)

type PaymentService struct {
	baseRepo         ledgerrepos.Repos
	ledgerService    *LedgerService
	providerRegistry *providers.ProviderRegistry
}

func NewPaymentService(baseRepo ledgerrepos.Repos, ledgerService *LedgerService, providerRegistry *providers.ProviderRegistry) *PaymentService {
	return &PaymentService{
		baseRepo:         baseRepo,
		ledgerService:    ledgerService,
		providerRegistry: providerRegistry,
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req *ledgerin.CreatePaymentRequest) (*ledgerout.CreatePaymentResponse, error) {
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

	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		if err := txRepos.PaymentRepository().CreateIntent(ctx, intent); err != nil {
			if errors.Is(err, ledgerrepo.ErrDuplicate) {
				return fmt.Errorf("%w: %s", ErrDuplicatePayment, req.TransactionID)
			}
			return err
		}
		return nil
	}); err != nil {
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

	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		persistedIntent, err := txRepos.PaymentRepository().GetIntentByTransactionID(ctx, req.TransactionID)
		if err != nil {
			if errors.Is(err, ledgerrepo.ErrNotFound) {
				return fmt.Errorf("%w: %s", ErrPaymentIntentNotFound, req.TransactionID)
			}
			return err
		}

		if err := txRepos.PaymentRepository().UpdateIntentProviderState(ctx, persistedIntent.TransactionID, response.ExternalRef, targetStatus); err != nil {
			return err
		}

		if targetStatus == entity.PaymentStatusSuccess {
			_, _, err := s.finalizeSuccessfulPayment(ctx, txRepos, persistedIntent, &providers.PaymentResult{
				TransactionID: response.TransactionID,
				Status:        targetStatus,
				Amount:        persistedIntent.Amount,
				Currency:      persistedIntent.Currency,
				ExternalRef:   response.ExternalRef,
			})
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	logging.FromContext(ctx).Infow("payment created",
		"provider", provider.Name(),
		"transaction_id", response.TransactionID,
		"status", targetStatus,
		"external_ref", response.ExternalRef,
	)

	return &ledgerout.CreatePaymentResponse{
		Provider:      strings.ToLower(provider.Name()),
		TransactionID: response.TransactionID,
		ExternalRef:   response.ExternalRef,
		Status:        targetStatus,
		CheckoutURL:   response.CheckoutURL,
	}, nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, providerName string, payload []byte, signature string) (*ledgerout.ProcessWebhookResponse, error) {
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

	var response *ledgerout.ProcessWebhookResponse
	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		intent, err := s.findIntent(ctx, txRepos.PaymentRepository(), strings.ToLower(provider.Name()), result)
		if err != nil {
			return err
		}
		if err := validateResultAgainstIntent(intent, result); err != nil {
			return err
		}

		if result.Status != entity.PaymentStatusSuccess {
			if err := txRepos.PaymentRepository().UpdateIntentProviderState(ctx, intent.TransactionID, result.ExternalRef, result.Status); err != nil {
				return err
			}
			response = &ledgerout.ProcessWebhookResponse{
				Provider:      intent.Provider,
				TransactionID: intent.TransactionID,
				ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
				Status:        result.Status,
			}
			return nil
		}

		ledgerPosted, duplicate, err := s.finalizeSuccessfulPayment(ctx, txRepos, intent, result)
		if err != nil {
			return err
		}

		response = &ledgerout.ProcessWebhookResponse{
			Provider:      intent.Provider,
			TransactionID: intent.TransactionID,
			ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
			Status:        entity.PaymentStatusSuccess,
			Duplicate:     duplicate,
			LedgerPosted:  ledgerPosted,
		}

		logging.FromContext(ctx).Infow("payment webhook processed",
			"provider", intent.Provider,
			"transaction_id", intent.TransactionID,
			"duplicate", duplicate,
			"ledger_posted", ledgerPosted,
		)

		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *PaymentService) finalizeSuccessfulPayment(ctx context.Context, txRepos ledgerrepos.Repos, intent *entity.PaymentIntent, result *providers.PaymentResult) (bool, bool, error) {
	idempotencyKey := paymentIdempotencyKey(intent, result)
	processed, err := txRepos.PaymentRepository().IsProcessed(ctx, intent.Provider, idempotencyKey)
	if err != nil {
		return false, false, err
	}
	if processed {
		if err := txRepos.PaymentRepository().UpdateIntentProviderState(ctx, intent.TransactionID, result.ExternalRef, entity.PaymentStatusSuccess); err != nil {
			return false, false, err
		}
		return false, true, nil
	}

	_, err = s.ledgerService.createTransaction(ctx, txRepos.LedgerRepository(), intent.TransactionID, []entity.LedgerEntryInput{
		{AccountID: intent.DebitAccountID, Amount: -intent.Amount},
		{AccountID: intent.CreditAccountID, Amount: intent.Amount},
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

	if err := txRepos.PaymentRepository().MarkProcessed(ctx, &entity.ProcessedPaymentEvent{
		Provider:       intent.Provider,
		IdempotencyKey: idempotencyKey,
		TransactionID:  intent.TransactionID,
		CreatedAt:      time.Now().UTC(),
	}); err != nil && !errors.Is(err, ledgerrepo.ErrDuplicate) {
		return false, false, err
	}

	if err := txRepos.PaymentRepository().UpdateIntentProviderState(ctx, intent.TransactionID, result.ExternalRef, entity.PaymentStatusSuccess); err != nil {
		return false, false, err
	}

	return ledgerPosted, duplicate, nil
}

func (s *PaymentService) findIntent(ctx context.Context, repo ledgerrepos.PaymentRepository, provider string, result *providers.PaymentResult) (*entity.PaymentIntent, error) {
	if strings.TrimSpace(result.TransactionID) != "" {
		intent, err := repo.GetIntentByTransactionID(ctx, result.TransactionID)
		if err == nil {
			return intent, nil
		}
		if !errors.Is(err, ledgerrepo.ErrNotFound) {
			return nil, err
		}
	}

	if strings.TrimSpace(result.ExternalRef) != "" {
		intent, err := repo.GetIntentByExternalRef(ctx, provider, result.ExternalRef)
		if err == nil {
			return intent, nil
		}
		if !errors.Is(err, ledgerrepo.ErrNotFound) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("%w: transaction_id=%s external_ref=%s", ErrPaymentIntentNotFound, result.TransactionID, result.ExternalRef)
}

func (s *PaymentService) updateIntentStatus(ctx context.Context, transactionID, status string) error {
	return s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		return txRepos.PaymentRepository().UpdateIntentStatus(ctx, transactionID, status)
	})
}

func validateResultAgainstIntent(intent *entity.PaymentIntent, result *providers.PaymentResult) error {
	if result.Amount != 0 && result.Amount != intent.Amount {
		return fmt.Errorf("%w: provider amount does not match reserved payment", ErrValidation)
	}
	if currency := strings.TrimSpace(result.Currency); currency != "" && !strings.EqualFold(currency, intent.Currency) {
		return fmt.Errorf("%w: provider currency does not match reserved payment", ErrValidation)
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
