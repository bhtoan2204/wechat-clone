package model

import "time"

type PaymentIntentModel struct {
	TransactionID   string    `gorm:"column:transaction_id;primaryKey"`
	Provider        string    `gorm:"column:provider;not null"`
	ExternalRef     *string   `gorm:"column:external_ref"`
	Amount          int64     `gorm:"column:amount;not null"`
	Currency        string    `gorm:"column:currency;not null"`
	DebitAccountID  string    `gorm:"column:debit_account_id;not null"`
	CreditAccountID string    `gorm:"column:credit_account_id;not null"`
	Status          string    `gorm:"column:status;not null"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (PaymentIntentModel) TableName() string {
	return "payment_intents"
}
