package events

import "time"

const (
	EventRoomMessageCreated = "EventRoomMessageCreated"
)

type RoomMessageMention struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name,omitempty"`
	Username    string `json:"username,omitempty"`
}

type RoomMessageCreatedEvent struct {
	RoomID                 string               `json:"room_id"`
	RoomName               string               `json:"room_name,omitempty"`
	RoomType               string               `json:"room_type,omitempty"`
	MessageID              string               `json:"message_id"`
	MessageContent         string               `json:"message_content,omitempty"`
	MessageType            string               `json:"message_type,omitempty"`
	ReplyToMessageID       string               `json:"reply_to_message_id,omitempty"`
	ForwardedFromMessageID string               `json:"forwarded_from_message_id,omitempty"`
	FileName               string               `json:"file_name,omitempty"`
	FileSize               int64                `json:"file_size,omitempty"`
	MimeType               string               `json:"mime_type,omitempty"`
	ObjectKey              string               `json:"object_key,omitempty"`
	MessageSenderID        string               `json:"message_sender_id"`
	MessageSenderName      string               `json:"message_sender_name,omitempty"`
	MessageSenderEmail     string               `json:"message_sender_email,omitempty"`
	MessageSentAt          time.Time            `json:"message_sent_at"`
	Mentions               []RoomMessageMention `json:"mentions,omitempty"`
	MentionAll             bool                 `json:"mention_all"`
	MentionedAccountIDs    []string             `json:"mentioned_account_ids,omitempty"`
}
