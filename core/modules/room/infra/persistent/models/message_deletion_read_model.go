package models

import "time"

type MessageDeletionReadModel struct {
	ID        string    `gorm:"primaryKey"`
	MessageID string    `gorm:"not null;index:idx_message_deletion_read_message_account,unique"`
	AccountID string    `gorm:"not null;index:idx_message_deletion_read_message_account,unique"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (MessageDeletionReadModel) TableName() string {
	return "message_deletion_read_models"
}
