package service

import (
	"context"

	"go-socket/core/modules/payment/domain/entity"
)

//go:generate mockgen -package=service -destination=payment_provider_mock.go -source=payment_provider.go
type PaymentProvider interface {
	Name() string
	CreatePayment(ctx context.Context, intent *entity.PaymentIntent, metadata map[string]string) (*PaymentCreation, error)
	ParseWebhook(ctx context.Context, payload []byte, signature string) (*PaymentWebhook, error)
}

//go:generate mockgen -package=service -destination=payment_provider_mock.go -source=payment_provider.go
type PaymentProviderRegistry interface {
	Get(name string) (PaymentProvider, error)
}

type PaymentCreation struct {
	Provider    string
	Result      entity.PaymentProviderResult
	CheckoutURL string
}

type PaymentWebhook struct {
	Provider string
	Ignored  bool
	Result   entity.PaymentProviderResult
}
