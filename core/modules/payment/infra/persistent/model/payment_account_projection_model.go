package model

import "time"

// This table is used to store the account projection data for the payment module.
type PaymentAccountProjectionModel struct {
	ID        string    `gorm:"primaryKey"`
	AccountID string    `gorm:"index"`
	Email     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (PaymentAccountProjectionModel) TableName() string {
	return "payment_account_projections"
}
