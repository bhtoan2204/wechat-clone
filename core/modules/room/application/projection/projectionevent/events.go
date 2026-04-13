package projectionevent

import (
	"time"

	sharedevents "go-socket/core/shared/contracts/events"
)

const (
	EventRoomProjectionUpserted                = "EventRoomProjectionUpserted"
	EventRoomProjectionDeleted                 = "EventRoomProjectionDeleted"
	EventRoomMemberProjectionUpserted          = "EventRoomMemberProjectionUpserted"
	EventRoomMemberProjectionDeleted           = "EventRoomMemberProjectionDeleted"
	EventRoomMessageProjectionUpserted         = "EventRoomMessageProjectionUpserted"
	EventRoomMessageReceiptProjectionUpserted  = "EventRoomMessageReceiptProjectionUpserted"
	EventRoomMessageDeletionProjectionUpserted = "EventRoomMessageDeletionProjectionUpserted"
)

type RoomUpserted struct {
	RoomID                 string     `json:"room_id"`
	Name                   string     `json:"name"`
	Description            string     `json:"description"`
	RoomType               string     `json:"room_type"`
	OwnerID                string     `json:"owner_id"`
	PinnedMessageID        string     `json:"pinned_message_id,omitempty"`
	MemberCount            int        `json:"member_count"`
	HasLastMessageSnapshot bool       `json:"has_last_message_snapshot"`
	LastMessageID          string     `json:"last_message_id,omitempty"`
	LastMessageAt          *time.Time `json:"last_message_at,omitempty"`
	LastMessageContent     string     `json:"last_message_content,omitempty"`
	LastMessageSenderID    string     `json:"last_message_sender_id,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type RoomDeleted struct {
	RoomID string `json:"room_id"`
}

type RoomMemberUpserted struct {
	RoomID          string     `json:"room_id"`
	MemberID        string     `json:"member_id"`
	AccountID       string     `json:"account_id"`
	Role            string     `json:"role"`
	LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
	LastReadAt      *time.Time `json:"last_read_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type RoomMemberDeleted struct {
	RoomID    string `json:"room_id"`
	AccountID string `json:"account_id"`
}

type RoomMessageUpserted struct {
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
	EditedAt               *time.Time                        `json:"edited_at,omitempty"`
	DeletedForEveryoneAt   *time.Time                        `json:"deleted_for_everyone_at,omitempty"`
}

type RoomMessageReceiptUpserted struct {
	RoomID      string     `json:"room_id"`
	MessageID   string     `json:"message_id"`
	AccountID   string     `json:"account_id"`
	Status      string     `json:"status"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	SeenAt      *time.Time `json:"seen_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type RoomMessageDeletionUpserted struct {
	RoomID        string    `json:"room_id"`
	MessageID     string    `json:"message_id"`
	AccountID     string    `json:"account_id"`
	MessageSentAt time.Time `json:"message_sent_at"`
	CreatedAt     time.Time `json:"created_at"`
}
