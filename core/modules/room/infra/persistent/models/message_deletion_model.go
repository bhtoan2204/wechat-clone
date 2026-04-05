package models

import "time"

type MessageDeletionModel struct {
	ID        string    `gorm:"primaryKey"`
	MessageID string    `gorm:"not null;index:idx_message_deletion_message_account,unique"`
	AccountID string    `gorm:"not null;index:idx_message_deletion_message_account,unique"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (MessageDeletionModel) TableName() string {
	return "message_deletions"
}
