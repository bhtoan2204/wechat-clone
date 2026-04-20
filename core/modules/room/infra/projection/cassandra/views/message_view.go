package views

import "time"

type MessageView struct {
	ID                     string
	RoomID                 string
	SenderID               string
	Message                string
	MessageType            string
	Mentions               []MessageMentionView
	Reactions              []MessageReactionView
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

type MessageReactionView struct {
	AccountID string
	Emoji     string
	ReactedAt time.Time
}
