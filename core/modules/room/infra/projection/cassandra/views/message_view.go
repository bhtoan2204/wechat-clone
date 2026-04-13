package views

import "time"

type MessageView struct {
	ID                     string
	RoomID                 string
	SenderID               string
	Message                string
	MessageType            string
	Mentions               []MessageMentionView
	MentionAll             bool
	ReplyToMessageID       string
	ForwardedFromMessageID string
	FileName               string
	FileSize               int64
	MimeType               string
	ObjectKey              string
	EditedAt               *time.Time
	DeletedForEveryoneAt   *time.Time
	CreatedAt              time.Time
}
