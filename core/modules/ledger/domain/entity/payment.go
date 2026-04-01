package entity

import "time"

const (
	PaymentStatusCreating = "CREATING"
	PaymentStatusPending  = "PENDING"
	PaymentStatusSuccess  = "SUCCESS"
	PaymentStatusFailed   = "FAILED"
)

type PaymentIntent struct {
	TransactionID   string
	Provider        string
	ExternalRef     string
	Amount          int64
	Currency        string
	DebitAccountID  string
	CreditAccountID string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ProcessedPaymentEvent struct {
	Provider       string
	IdempotencyKey string
	TransactionID  string
	CreatedAt      time.Time
}
