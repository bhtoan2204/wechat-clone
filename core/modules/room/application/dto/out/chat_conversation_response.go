// CODE_GENERATOR: response
package out

type ChatConversationResponse struct {
	RoomID          string                   `json:"room_id"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	RoomType        string                   `json:"room_type"`
	OwnerID         string                   `json:"owner_id"`
	PinnedMessageID string                   `json:"pinned_message_id"`
	MemberCount     int                      `json:"member_count"`
	UnreadCount     int64                    `json:"unread_count"`
	LastMessage     *ChatMessageResponse     `json:"last_message"`
	Members         []ChatRoomMemberResponse `json:"members"`
	CreatedAt       string                   `json:"created_at"`
	UpdatedAt       string                   `json:"updated_at"`
}
