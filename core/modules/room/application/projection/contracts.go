package projection

import (
	"context"
	"time"
)

type TimelineProjector interface {
	UpsertMessage(ctx context.Context, projection *TimelineMessageProjection) error
}

type MessageSearchIndexer interface {
	UpsertMessage(ctx context.Context, document *SearchMessageDocument) error
}

type ProjectionMention struct {
	AccountID   string
	DisplayName string
	Username    string
}

type TimelineMessageProjection struct {
	RoomID                 string
	RoomName               string
	RoomType               string
	MessageID              string
	MessageContent         string
	MessageType            string
	ReplyToMessageID       string
	ForwardedFromMessageID string
	FileName               string
	FileSize               int64
	MimeType               string
	ObjectKey              string
	MessageSenderID        string
	MessageSenderName      string
	MessageSenderEmail     string
	MessageSentAt          time.Time
	Mentions               []ProjectionMention
	MentionAll             bool
	MentionedAccountIDs    []string
}

type SearchMessageDocument struct {
	RoomID                 string
	RoomName               string
	RoomType               string
	MessageID              string
	MessageContent         string
	MessageType            string
	ReplyToMessageID       string
	ForwardedFromMessageID string
	FileName               string
	FileSize               int64
	MimeType               string
	ObjectKey              string
	MessageSenderID        string
	MessageSenderName      string
	MessageSenderEmail     string
	MessageSentAt          time.Time
	Mentions               []ProjectionMention
	MentionAll             bool
	MentionedAccountIDs    []string
}
