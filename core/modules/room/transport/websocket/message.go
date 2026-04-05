package socket

import "encoding/json"

const (
	ActionJoinRoom    = "JOIN_ROOM"
	ActionLeaveRoom   = "LEAVE_ROOM"
	ActionChatMessage = "CHAT_MESSAGE"
	ActionTyping      = "TYPING"
	ActionPresence    = "PRESENCE"
	ActionSeen        = "SEEN"
)

type Message struct {
	Action   string          `json:"action"`
	RoomID   string          `json:"room_id,omitempty"`
	SenderID string          `json:"sender_id,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}
