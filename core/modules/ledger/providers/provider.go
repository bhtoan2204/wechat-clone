package providers

import (
	"context"
	"errors"
)

var (
	ErrProviderNotFound        = errors.New("provider not found")
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
)

type CreatePaymentRequest struct {
	TransactionID   string
	Amount          int64
	Currency        string
	DebitAccountID  string
	CreditAccountID string
	Metadata        map[string]string
}

type CreatePaymentResponse struct {
	Provider      string `json:"provider"`
	TransactionID string `json:"transaction_id"`
	ExternalRef   string `json:"external_ref,omitempty"`
	Status        string `json:"status"`
	CheckoutURL   string `json:"checkout_url,omitempty"`
}

type WebhookEvent struct {
	Provider   string
	EventID    string
	EventType  string
	Attributes map[string]string
}

type PaymentResult struct {
	TransactionID string
	Status        string
	Amount        int64
	Currency      string
	ExternalRef   string
}

type PaymentProvider interface {
	Name() string
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error)
	VerifyWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
	ParseEvent(ctx context.Context, event *WebhookEvent) (*PaymentResult, error)
}
