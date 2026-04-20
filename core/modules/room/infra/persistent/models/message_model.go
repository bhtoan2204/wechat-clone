package models

import "time"

type MessageModel struct {
	ID                     string     `gorm:"primaryKey" json:"id"`
	RoomID                 string     `gorm:"not null;index" json:"room_id"`
	SenderID               string     `gorm:"not null;index" json:"sender_id"`
	Message                string     `gorm:"type:VARCHAR2(4000)" json:"message"`
	MessageType            string     `gorm:"type:VARCHAR2(50);default:'text';not null" json:"message_type"`
	MentionsJSON           string     `gorm:"type:CLOB;not null;default:'[]'" json:"mentions_json"`
	ReactionsJSON          string     `gorm:"type:CLOB;not null;default:'[]'" json:"reactions_json"`
	MentionAll             bool       `gorm:"type:number(1);default:0;not null" json:"mention_all"`
	ReplyToMessageID       *string    `gorm:"index" json:"reply_to_message_id"`
	ForwardedFromMessageID *string    `gorm:"index" json:"forwarded_from_message_id"`
	FileName               *string    `gorm:"type:VARCHAR2(1024)" json:"file_name"`
	FileSize               *int64     `json:"file_size"`
	MimeType               *string    `gorm:"type:VARCHAR2(255)" json:"mime_type"`
	ObjectKey              *string    `gorm:"type:VARCHAR2(2048)" json:"object_key"`
	EditedAt               *time.Time `json:"edited_at"`
	DeletedForEveryoneAt   *time.Time `json:"deleted_for_everyone_at"`
	CreatedAt              time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (MessageModel) TableName() string {
	return "messages"
}
