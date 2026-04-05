package models

import "time"

type MessageReceiptModel struct {
	ID          string `gorm:"primaryKey"`
	MessageID   string `gorm:"not null;index:idx_message_receipt_message_account,unique"`
	AccountID   string `gorm:"not null;index:idx_message_receipt_message_account,unique"`
	Status      string `gorm:"type:VARCHAR2(32);not null"`
	DeliveredAt *time.Time
	SeenAt      *time.Time
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (MessageReceiptModel) TableName() string {
	return "message_receipts"
}
