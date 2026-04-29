package aggregate

import (
	"time"
	"wechat-clone/core/modules/room/types"
	sharedevents "wechat-clone/core/shared/contracts/events"
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

type EventRoomOwnerChanged struct {
	RoomID          string    `json:"room_id"`
	PreviousOwnerID string    `json:"previous_owner_id,omitempty"`
	OwnerID         string    `json:"owner_id"`
	ChangedAt       time.Time `json:"changed_at"`
}

type EventRoomDetailsUpdated struct {
	RoomID      string         `json:"room_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	RoomType    types.RoomType `json:"room_type"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type EventRoomMessagePinned struct {
	RoomID          string    `json:"room_id"`
	PinnedMessageID string    `json:"pinned_message_id"`
	PinnedAt        time.Time `json:"pinned_at"`
}

type EventRoomMemberAdded struct {
	RoomID               string         `json:"room_id"`
	MemberID             string         `json:"member_id"`
	RoomMemberID         string         `json:"room_member_id,omitempty"`
	MemberName           string         `json:"member_name,omitempty"`
	MemberEmail          string         `json:"member_email,omitempty"`
	MemberUsername       string         `json:"member_username,omitempty"`
	MemberAvatarKey      string         `json:"member_avatar_key,omitempty"`
	MemberRole           types.RoomRole `json:"member_role"`
	MemberJoinedAt       time.Time      `json:"member_joined_at"`
	MemberStateUpdatedAt time.Time      `json:"member_state_updated_at,omitempty"`
}

type EventRoomMemberRemoved struct {
	RoomID         string         `json:"room_id"`
	MemberID       string         `json:"member_id"`
	RoomMemberID   string         `json:"room_member_id,omitempty"`
	MemberName     string         `json:"member_name,omitempty"`
	MemberEmail    string         `json:"member_email,omitempty"`
	MemberUsername string         `json:"member_username,omitempty"`
	MemberRole     types.RoomRole `json:"member_role"`
	MemberJoinedAt time.Time      `json:"member_joined_at"`
	RemovedAt      time.Time      `json:"removed_at"`
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
