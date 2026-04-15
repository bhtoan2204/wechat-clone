package events

import "time"

const (
	EventPaymentCreated                = "payment.created"
	EventPaymentCheckoutSessionCreated = "payment.checkout_session_created"
	EventPaymentSucceeded              = "payment.succeeded"
	EventPaymentFailed                 = "payment.failed"
)

type PaymentCreatedEvent struct {
	PaymentID          string            `json:"payment_id"`
	TransactionID      string            `json:"transaction_id"`
	Provider           string            `json:"provider"`
	ClearingAccountKey string            `json:"clearing_account_key"`
	Amount             int64             `json:"amount"`
	Currency           string            `json:"currency"`
	CreditAccountID    string            `json:"credit_account_id"`
	Status             string            `json:"status"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
}

type PaymentCheckoutSessionCreatedEvent struct {
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	CheckoutURL        string    `json:"checkout_url,omitempty"`
	Amount             int64     `json:"amount"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	OccurredAt         time.Time `json:"occurred_at"`
}

type PaymentSucceededEvent struct {
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ClearingAccountKey string    `json:"clearing_account_key"`
	ProviderEventID    string    `json:"provider_event_id,omitempty"`
	ProviderEventType  string    `json:"provider_event_type,omitempty"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	Amount             int64     `json:"amount"`
	Currency           string    `json:"currency"`
	CreditAccountID    string    `json:"credit_account_id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	SucceededAt        time.Time `json:"succeeded_at"`
}

type PaymentFailedEvent struct {
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ProviderEventID    string    `json:"provider_event_id,omitempty"`
	ProviderEventType  string    `json:"provider_event_type,omitempty"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	Amount             int64     `json:"amount"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	OccurredAt         time.Time `json:"occurred_at"`
}
