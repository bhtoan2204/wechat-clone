package provider

import (
	"context"
	"strings"

	"go-socket/core/modules/payment/domain/entity"
	domainservice "go-socket/core/modules/payment/domain/service"
	"go-socket/core/modules/payment/providers"
	"go-socket/core/shared/pkg/stackErr"
)

type paymentProviderRegistry struct {
	registry *providers.ProviderRegistry
}

type paymentProviderAdapter struct {
	provider providers.PaymentProvider
}

func NewPaymentProviderRegistry(registry *providers.ProviderRegistry) domainservice.PaymentProviderRegistry {
	return &paymentProviderRegistry{registry: registry}
}

func (r *paymentProviderRegistry) Get(name string) (domainservice.PaymentProvider, error) {
	provider, err := r.registry.Get(name)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &paymentProviderAdapter{provider: provider}, nil
}

func (a *paymentProviderAdapter) Name() string {
	return strings.ToLower(strings.TrimSpace(a.provider.Name()))
}

func (a *paymentProviderAdapter) CreatePayment(
	ctx context.Context,
	intent *entity.PaymentIntent,
	metadata map[string]string,
) (*domainservice.PaymentCreation, error) {
	response, err := a.provider.CreatePayment(ctx, providers.CreatePaymentRequest{
		TransactionID:   intent.TransactionID,
		Amount:          intent.Amount,
		Currency:        intent.Currency,
		DebitAccountID:  intent.DebitAccountID,
		CreditAccountID: intent.CreditAccountID,
		Metadata:        metadata,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &domainservice.PaymentCreation{
		Provider: a.Name(),
		Result: entity.PaymentProviderResult{
			TransactionID: coalesceProviderValue(response.TransactionID, intent.TransactionID),
			Status:        entity.NormalizePaymentStatusOrPending(response.Status),
			ExternalRef:   strings.TrimSpace(response.ExternalRef),
		},
		CheckoutURL: strings.TrimSpace(response.CheckoutURL),
	}, nil
}

func (a *paymentProviderAdapter) ParseWebhook(
	ctx context.Context,
	payload []byte,
	signature string,
) (*domainservice.PaymentWebhook, error) {
	event, err := a.provider.VerifyWebhook(ctx, payload, signature)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	result, err := a.provider.ParseEvent(ctx, event)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &domainservice.PaymentWebhook{
		Provider: a.Name(),
		Result: entity.PaymentProviderResult{
			TransactionID: strings.TrimSpace(result.TransactionID),
			EventID:       strings.TrimSpace(result.EventID),
			EventType:     strings.TrimSpace(result.EventType),
			Status:        entity.NormalizePaymentStatusOrPending(result.Status),
			Amount:        result.Amount,
			Currency:      strings.TrimSpace(result.Currency),
			ExternalRef:   strings.TrimSpace(result.ExternalRef),
		},
	}, nil
}

func coalesceProviderValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
