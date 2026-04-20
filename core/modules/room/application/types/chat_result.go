package types

type PresenceResult struct {
	AccountID string
	Status    string
}

type ConversationMemberResult struct {
	AccountID       string
	Role            string
	DisplayName     string
	Username        string
	AvatarObjectKey string
}

type MessagePreviewResult struct {
	ID          string
	SenderID    string
	Message     string
	MessageType string
}

type MessageMentionResult struct {
	AccountID   string
	DisplayName string
	Username    string
}

type MessageReactionResult struct {
	Emoji       string
	Count       int
	ReactedByMe bool
	AccountIDs  []string
}

type MentionCandidateResult struct {
	AccountID       string
	DisplayName     string
	Username        string
	AvatarObjectKey string
}

type MessageResult struct {
	ID                     string
	RoomID                 string
	SenderID               string
	Message                string
	MessageType            string
	Status                 string
	Mentions               []MessageMentionResult
	Reactions              []MessageReactionResult
	MentionAll             bool
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
