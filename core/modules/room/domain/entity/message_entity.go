package entity

import "time"

type MessageEntity struct {
	ID                     string     `json:"id"`
	RoomID                 string     `json:"room_id"`
	SenderID               string     `json:"sender_id"`
	Message                string     `json:"message"`
	MessageType            string     `json:"message_type"`
	ReplyToMessageID       string     `json:"reply_to_message_id,omitempty"`
	ForwardedFromMessageID string     `json:"forwarded_from_message_id,omitempty"`
	FileName               string     `json:"file_name,omitempty"`
	FileSize               int64      `json:"file_size,omitempty"`
	MimeType               string     `json:"mime_type,omitempty"`
	ObjectKey              string     `json:"object_key,omitempty"`
	EditedAt               *time.Time `json:"edited_at,omitempty"`
	DeletedForEveryoneAt   *time.Time `json:"deleted_for_everyone_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
}
