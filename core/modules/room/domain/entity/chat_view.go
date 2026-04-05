package entity

import "time"

type ConversationView struct {
	RoomID          string
	Name            string
	Description     string
	RoomType        string
	OwnerID         string
	PinnedMessageID string
	MemberCount     int
	UnreadCount     int64
	LastMessage     *MessageView
	Members         []ConversationMemberView
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ConversationMemberView struct {
	AccountID string
	Role      string
}

type MessageView struct {
	ID                     string
	RoomID                 string
	SenderID               string
	Message                string
	MessageType            string
	Status                 string
	ReplyToMessageID       string
	ForwardedFromMessageID string
	FileName               string
	FileSize               int64
	MimeType               string
	ObjectKey              string
	EditedAt               *time.Time
	DeletedForEveryone     bool
	CreatedAt              time.Time
	ReplyTo                *MessagePreviewView
	ForwardedFrom          *MessagePreviewView
}

type MessagePreviewView struct {
	ID          string
	SenderID    string
	Message     string
	MessageType string
}
