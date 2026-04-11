package aggregate

import (
	"go-socket/core/modules/room/types"
	sharedevents "go-socket/core/shared/contracts/events"
	"time"
)

type EventRoomCreated struct {
	RoomID              string         `json:"room_id"`
	RoomType            types.RoomType `json:"room_type"`
	MemberCount         int            `json:"member_count"`
	LastMessageID       string         `json:"last_message_id,omitempty"`
	LastMessageAt       time.Time      `json:"last_message_at,omitempty"`
	LastMessageContent  string         `json:"last_message_content,omitempty"`
	LastMessageSenderID string         `json:"last_message_sender_id,omitempty"`
}

type EventRoomMemberAdded struct {
	RoomID         string         `json:"room_id"`
	MemberID       string         `json:"member_id"`
	MemberName     string         `json:"member_name,omitempty"`
	MemberEmail    string         `json:"member_email,omitempty"`
	MemberRole     types.RoomRole `json:"member_role"`
	MemberJoinedAt time.Time      `json:"member_joined_at"`
}

type EventRoomMemberRemoved struct {
	RoomID         string         `json:"room_id"`
	MemberID       string         `json:"member_id"`
	MemberName     string         `json:"member_name,omitempty"`
	MemberEmail    string         `json:"member_email,omitempty"`
	MemberRole     types.RoomRole `json:"member_role"`
	MemberJoinedAt time.Time      `json:"member_joined_at"`
}

type EventRoomMessageCreated struct {
	RoomID                 string                            `json:"room_id"`
	RoomName               string                            `json:"room_name,omitempty"`
	RoomType               string                            `json:"room_type,omitempty"`
	MessageID              string                            `json:"message_id"`
	MessageContent         string                            `json:"message_content,omitempty"`
	MessageType            string                            `json:"message_type,omitempty"`
	ReplyToMessageID       string                            `json:"reply_to_message_id,omitempty"`
	ForwardedFromMessageID string                            `json:"forwarded_from_message_id,omitempty"`
	FileName               string                            `json:"file_name,omitempty"`
	FileSize               int64                             `json:"file_size,omitempty"`
	MimeType               string                            `json:"mime_type,omitempty"`
	ObjectKey              string                            `json:"object_key,omitempty"`
	MessageSenderID        string                            `json:"message_sender_id"`
	MessageSenderName      string                            `json:"message_sender_name,omitempty"`
	MessageSenderEmail     string                            `json:"message_sender_email,omitempty"`
	MessageSentAt          time.Time                         `json:"message_sent_at"`
	Mentions               []sharedevents.RoomMessageMention `json:"mentions,omitempty"`
	MentionAll             bool                              `json:"mention_all"`
	MentionedAccountIDs    []string                          `json:"mentioned_account_ids,omitempty"`
}
