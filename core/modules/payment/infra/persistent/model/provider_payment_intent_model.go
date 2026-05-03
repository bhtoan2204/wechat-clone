package model

import "time"

type PaymentIntentModel struct {
	TransactionID        string `gorm:"primaryKey"`
	Workflow             string `gorm:"not null"`
	Provider             string `gorm:"not null"`
	ExternalRef          *string
	DestinationAccountID *string   `gorm:"column:destination_account_id"`
	Amount               int64     `gorm:"not null"`
	FeeAmount            int64     `gorm:"not null;default:0"`
	ProviderAmount       int64     `gorm:"column:provider_amount;not null;default:0"`
	Currency             string    `gorm:"not null"`
	ClearingAccountKey   string    `gorm:"column:clearing_account_key;not null"`
	DebitAccountID       *string   `gorm:"column:debit_account_id"`
	CreditAccountID      *string   `gorm:"column:credit_account_id"`
	Status               string    `gorm:"not null"`
	Version              int       `gorm:"not null;default:0"`
	CreatedAt            time.Time `gorm:"autoCreateTime"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime"`
}

func (PaymentIntentModel) TableName() string {
	return "payment_intents"
}
