package model

import "time"

type ProviderPaymentIntentModel struct {
	TransactionID      string `gorm:"primaryKey"`
	Provider           string `gorm:"not null"`
	ExternalRef        *string
	Amount             int64     `gorm:"not null"`
	Currency           string    `gorm:"not null"`
	ClearingAccountKey string    `gorm:"column:clearing_account_key;not null"`
	CreditAccountID    string    `gorm:"not null"`
	Status             string    `gorm:"not null"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}

func (ProviderPaymentIntentModel) TableName() string {
	return "payment_intents"
}
