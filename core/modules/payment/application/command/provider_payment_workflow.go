package command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	paymentservice "go-socket/core/modules/payment/application/service"
	"go-socket/core/modules/payment/domain/entity"
	repos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/providers"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

func finalizeSuccessfulPayment(ctx context.Context, baseRepo repos.Repos, store repos.ProviderPaymentRepository, intent *entity.PaymentIntent, result entity.PaymentProviderResult) (bool, error) {
	idempotencyKey := intent.PaymentIdempotencyKey(result.EventID, result.ExternalRef)
	processed, err := store.IsProcessed(ctx, intent.Provider, idempotencyKey)
	if err != nil {
		return false, err
	}
	if processed {
		if err := intent.ApplyProviderResult(result, time.Now().UTC()); err != nil {
			return false, err
		}
		if err := store.SavePaymentIntent(ctx, intent); err != nil {
			return false, err
		}
		return true, nil
	}

	if err := baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return finalizeSuccessfulPaymentTx(ctx, tx.ProviderPaymentRepository(), intent, result)
	}); err != nil {
		if errors.Is(err, repos.ErrProviderPaymentDuplicateProcessed) {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

func finalizeSuccessfulPaymentTx(
	ctx context.Context,
	store repos.ProviderPaymentRepository,
	intent *entity.PaymentIntent,
	result entity.PaymentProviderResult,
	outboxEvents ...eventpkg.Event,
) error {
	updatedAt := time.Now().UTC()
	if err := intent.ApplyProviderResult(result, updatedAt); err != nil {
		return stackErr.Error(err)
	}

	processedEvent, err := intent.NewProcessedEvent(result, updatedAt)
	if err != nil {
		return stackErr.Error(err)
	}

	if err := store.FinalizeSuccessfulPayment(
		ctx,
		intent,
		processedEvent,
		intent.SucceededEvent(result, updatedAt),
		outboxEvents...,
	); err != nil {
		if errors.Is(err, repos.ErrProviderPaymentDuplicateProcessed) {
			return err
		}
		return stackErr.Error(err)
	}

	return nil
}

func findIntent(ctx context.Context, store repos.ProviderPaymentRepository, provider string, result *providers.PaymentResult) (*entity.PaymentIntent, error) {
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

	return nil, fmt.Errorf("%v: transaction_id=%s external_ref=%s", paymentservice.ErrPaymentIntentNotFound, result.TransactionID, result.ExternalRef)
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
	return stackErr.Error(fmt.Errorf("%v: %s", paymentservice.ErrValidation, err.Error()))
}

func toDomainPaymentResult(result *providers.PaymentResult) entity.PaymentProviderResult {
	if result == nil {
		return entity.PaymentProviderResult{}
	}
	return entity.PaymentProviderResult{
		TransactionID: result.TransactionID,
		EventID:       result.EventID,
		EventType:     result.EventType,
		Status:        entity.NormalizePaymentStatusOrPending(result.Status),
		Amount:        result.Amount,
		Currency:      result.Currency,
		ExternalRef:   result.ExternalRef,
	}
}
