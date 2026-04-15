package entity

import "time"

const (
	PaymentAggregateType  = "payment"
	PaymentStatusCreating = "CREATING"
	PaymentStatusPending  = "PENDING"
	PaymentStatusSuccess  = "SUCCESS"
	PaymentStatusFailed   = "FAILED"
)

type PaymentIntent struct {
	TransactionID      string
	Provider           string
	ExternalRef        string
	Amount             int64
	Currency           string
	ClearingAccountKey string
	CreditAccountID    string
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type PaymentProviderResult struct {
	TransactionID string
	EventID       string
	EventType     string
	Status        string
	Amount        int64
	Currency      string
	ExternalRef   string
}

type ProcessedPaymentEvent struct {
	Provider       string
	IdempotencyKey string
	TransactionID  string
	CreatedAt      time.Time
}
