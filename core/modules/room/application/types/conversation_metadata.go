package types

type ConversationMetadataPeerResult struct {
	AccountID       string
	DisplayName     string
	Username        string
	AvatarObjectKey string
}

type ConversationMetadataResult struct {
	RoomID                string
	RoomType              string
	OwnerID               string
	MemberCount           int
	PinnedMessageID       string
	LastMessageID         string
	ViewerRole            string
	ViewerLastDeliveredAt string
	ViewerLastReadAt      string
	IsOwner               bool
	DirectPeer            *ConversationMetadataPeerResult
}
