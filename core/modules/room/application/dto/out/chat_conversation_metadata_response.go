// CODE_GENERATOR - do not edit: response
package out

type ChatConversationMetadataResponse struct {
	RoomID                string                                `json:"room_id,omitempty"`
	RoomType              string                                `json:"room_type,omitempty"`
	OwnerID               string                                `json:"owner_id,omitempty"`
	MemberCount           int                                   `json:"member_count,omitempty"`
	PinnedMessageID       string                                `json:"pinned_message_id,omitempty"`
	LastMessageID         string                                `json:"last_message_id,omitempty"`
	ViewerRole            string                                `json:"viewer_role,omitempty"`
	ViewerLastDeliveredAt string                                `json:"viewer_last_delivered_at,omitempty"`
	ViewerLastReadAt      string                                `json:"viewer_last_read_at,omitempty"`
	IsOwner               bool                                  `json:"is_owner,omitempty"`
	DirectPeer            *ChatConversationMetadataPeerResponse `json:"direct_peer,omitempty"`
}
