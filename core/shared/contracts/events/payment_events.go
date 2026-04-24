package events

import (
	"time"

	"wechat-clone/core/shared/pkg/event"
)

type PaymentCreatedEvent struct {
	Workflow           string            `json:"workflow"`
	PaymentID          string            `json:"payment_id"`
	TransactionID      string            `json:"transaction_id"`
	Provider           string            `json:"provider"`
	ClearingAccountKey string            `json:"clearing_account_key"`
	DebitAccountID     string            `json:"debit_account_id,omitempty"`
	Amount             int64             `json:"amount"`
	FeeAmount          int64             `json:"fee_amount"`
	ProviderAmount     int64             `json:"provider_amount"`
	Currency           string            `json:"currency"`
	CreditAccountID    string            `json:"credit_account_id,omitempty"`
	Status             string            `json:"status"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
}

type PaymentWithdrawalRequestedEvent struct {
	PaymentID            string    `json:"payment_id"`
	TransactionID        string    `json:"transaction_id"`
	Provider             string    `json:"provider"`
	ClearingAccountKey   string    `json:"clearing_account_key"`
	DebitAccountID       string    `json:"debit_account_id"`
	DestinationAccountID string    `json:"destination_account_id"`
	Amount               int64     `json:"amount"`
	FeeAmount            int64     `json:"fee_amount"`
	ProviderAmount       int64     `json:"provider_amount"`
	Currency             string    `json:"currency"`
	Status               string    `json:"status"`
	RequestedAt          time.Time `json:"requested_at"`
}

type PaymentCheckoutSessionCreatedEvent struct {
	Workflow           string    `json:"workflow"`
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	CheckoutURL        string    `json:"checkout_url,omitempty"`
	Amount             int64     `json:"amount"`
	FeeAmount          int64     `json:"fee_amount"`
	ProviderAmount     int64     `json:"provider_amount"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	OccurredAt         time.Time `json:"occurred_at"`
}

type PaymentSucceededEvent struct {
	Workflow           string    `json:"workflow"`
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ClearingAccountKey string    `json:"clearing_account_key"`
	DebitAccountID     string    `json:"debit_account_id,omitempty"`
	ProviderEventID    string    `json:"provider_event_id,omitempty"`
	ProviderEventType  string    `json:"provider_event_type,omitempty"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	Amount             int64     `json:"amount"`
	FeeAmount          int64     `json:"fee_amount"`
	ProviderAmount     int64     `json:"provider_amount"`
	Currency           string    `json:"currency"`
	CreditAccountID    string    `json:"credit_account_id,omitempty"`
	IdempotencyKey     string    `json:"idempotency_key"`
	SucceededAt        time.Time `json:"succeeded_at"`
}

type PaymentFailedEvent struct {
	Workflow           string    `json:"workflow"`
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ClearingAccountKey string    `json:"clearing_account_key"`
	DebitAccountID     string    `json:"debit_account_id,omitempty"`
	ProviderEventID    string    `json:"provider_event_id,omitempty"`
	ProviderEventType  string    `json:"provider_event_type,omitempty"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	Amount             int64     `json:"amount"`
	FeeAmount          int64     `json:"fee_amount"`
	ProviderAmount     int64     `json:"provider_amount"`
	Currency           string    `json:"currency"`
	CreditAccountID    string    `json:"credit_account_id,omitempty"`
	Status             string    `json:"status"`
	OccurredAt         time.Time `json:"occurred_at"`
}

type PaymentRefundedEvent struct {
	Workflow           string    `json:"workflow"`
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ClearingAccountKey string    `json:"clearing_account_key"`
	DebitAccountID     string    `json:"debit_account_id,omitempty"`
	ProviderEventID    string    `json:"provider_event_id,omitempty"`
	ProviderEventType  string    `json:"provider_event_type,omitempty"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	Amount             int64     `json:"amount"`
	FeeAmount          int64     `json:"fee_amount"`
	ProviderAmount     int64     `json:"provider_amount"`
	Currency           string    `json:"currency"`
	CreditAccountID    string    `json:"credit_account_id,omitempty"`
	IdempotencyKey     string    `json:"idempotency_key"`
	RefundedAt         time.Time `json:"refunded_at"`
}

type PaymentChargebackEvent struct {
	Workflow           string    `json:"workflow"`
	PaymentID          string    `json:"payment_id"`
	TransactionID      string    `json:"transaction_id"`
	Provider           string    `json:"provider"`
	ClearingAccountKey string    `json:"clearing_account_key"`
	DebitAccountID     string    `json:"debit_account_id,omitempty"`
	ProviderEventID    string    `json:"provider_event_id,omitempty"`
	ProviderEventType  string    `json:"provider_event_type,omitempty"`
	ProviderPaymentRef string    `json:"provider_payment_ref,omitempty"`
	Amount             int64     `json:"amount"`
	FeeAmount          int64     `json:"fee_amount"`
	ProviderAmount     int64     `json:"provider_amount"`
	Currency           string    `json:"currency"`
	CreditAccountID    string    `json:"credit_account_id,omitempty"`
	IdempotencyKey     string    `json:"idempotency_key"`
	ChargedBackAt      time.Time `json:"charged_back_at"`
}

var (
	EventPaymentCreated                = event.EventName((*PaymentCreatedEvent)(nil))
	EventPaymentWithdrawalRequested    = event.EventName((*PaymentWithdrawalRequestedEvent)(nil))
	EventPaymentCheckoutSessionCreated = event.EventName((*PaymentCheckoutSessionCreatedEvent)(nil))
	EventPaymentSucceeded              = event.EventName((*PaymentSucceededEvent)(nil))
	EventPaymentFailed                 = event.EventName((*PaymentFailedEvent)(nil))
	EventPaymentRefunded               = event.EventName((*PaymentRefundedEvent)(nil))
	EventPaymentChargeback             = event.EventName((*PaymentChargebackEvent)(nil))
)
