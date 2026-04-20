package entity

import "time"

type MessageEntity struct {
	ID                     string
	RoomID                 string
	SenderID               string
	Message                string
	MessageType            string
	Mentions               []MessageMention
	Reactions              []MessageReaction
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
