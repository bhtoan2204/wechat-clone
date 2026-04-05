package types

type PresenceResult struct {
	AccountID string
	Status    string
}

type ConversationMemberResult struct {
	AccountID string
	Role      string
}

type MessagePreviewResult struct {
	ID          string
	SenderID    string
	Message     string
	MessageType string
}

type MessageResult struct {
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
	EditedAt               string
	DeletedForEveryone     bool
	CreatedAt              string
	ReplyTo                *MessagePreviewResult
	ForwardedFrom          *MessagePreviewResult
}

type ConversationResult struct {
	RoomID          string
	Name            string
	Description     string
	RoomType        string
	OwnerID         string
	PinnedMessageID string
	MemberCount     int
	UnreadCount     int64
	LastMessage     *MessageResult
	Members         []ConversationMemberResult
	CreatedAt       string
	UpdatedAt       string
}
