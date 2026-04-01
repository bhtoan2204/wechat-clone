package model

import "time"

type ProcessedPaymentEventModel struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Provider       string    `gorm:"column:provider;not null"`
	IdempotencyKey string    `gorm:"column:idempotency_key;not null"`
	TransactionID  string    `gorm:"column:transaction_id;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ProcessedPaymentEventModel) TableName() string {
	return "processed_payment_events"
}
