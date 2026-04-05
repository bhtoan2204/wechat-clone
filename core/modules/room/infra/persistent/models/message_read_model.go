package models

import "time"

type MessageReadModel struct {
	ID                     string  `gorm:"primaryKey"`
	RoomID                 string  `gorm:"not null;index"`
	SenderID               string  `gorm:"not null;index"`
	Message                string  `gorm:"type:VARCHAR2(4000);not null"`
	MessageType            string  `gorm:"type:VARCHAR2(50);default:'text';not null"`
	ReplyToMessageID       *string `gorm:"index"`
	ForwardedFromMessageID *string `gorm:"index"`
	FileName               *string `gorm:"type:VARCHAR2(1024)"`
	FileSize               *int64
	MimeType               *string `gorm:"type:VARCHAR2(255)"`
	ObjectKey              *string `gorm:"type:VARCHAR2(2048)"`
	EditedAt               *time.Time
	DeletedForEveryoneAt   *time.Time
	CreatedAt              time.Time `gorm:"autoCreateTime"`
}

func (MessageReadModel) TableName() string {
	return "message_read_models"
}
