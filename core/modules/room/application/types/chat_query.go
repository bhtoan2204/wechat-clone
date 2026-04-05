package types

type ListConversationsQuery struct {
	Limit  int
	Offset int
}

type GetConversationQuery struct {
	RoomID string
}

type ListMessagesQuery struct {
	RoomID    string
	Limit     int
	BeforeID  string
	BeforeAt  string
	Ascending bool
}

type GetPresenceQuery struct {
	AccountID string
}
