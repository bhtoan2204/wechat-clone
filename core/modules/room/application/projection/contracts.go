package projection

import (
	"context"
	"time"
)

const (
	EventRoomAggregateProjectionSynced    = "EventRoomAggregateProjectionSynced"
	EventRoomAggregateProjectionDeleted   = "EventRoomAggregateProjectionDeleted"
	EventMessageAggregateProjectionSynced = "EventMessageAggregateProjectionSynced"
)

//go:generate mockgen -package=projection -destination=contracts_mock.go -source=contracts.go
type ServingProjector interface {
	SyncRoomAggregate(ctx context.Context, projection *RoomAggregateSync) error
	DeleteRoomAggregate(ctx context.Context, roomID string) error
	SyncMessageAggregate(ctx context.Context, projection *MessageAggregateSync) error
}

//go:generate mockgen -package=projection -destination=contracts_mock.go -source=contracts.go
type MessageSearchIndexer interface {
	SyncMessage(ctx context.Context, message *MessageProjection) error
	DeleteRoom(ctx context.Context, roomID string) error
}

type ProjectionMention struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}

type RoomAggregateDeleted struct {
	RoomID string `json:"room_id"`
}

type RoomAggregateSync struct {
	Room    *RoomProjection        `json:"room,omitempty"`
	Members []RoomMemberProjection `json:"members,omitempty"`
}

type RoomProjection struct {
	RoomID          string                     `json:"room_id"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description"`
	RoomType        string                     `json:"room_type"`
	OwnerID         string                     `json:"owner_id"`
	PinnedMessageID string                     `json:"pinned_message_id,omitempty"`
	MemberCount     int                        `json:"member_count"`
	LastMessage     *RoomLastMessageProjection `json:"last_message,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

type RoomLastMessageProjection struct {
	MessageID       string     `json:"message_id"`
	MessageSentAt   *time.Time `json:"message_sent_at,omitempty"`
	MessageContent  string     `json:"message_content,omitempty"`
	MessageSenderID string     `json:"message_sender_id,omitempty"`
}

type RoomMemberProjection struct {
	RoomID          string     `json:"room_id"`
	MemberID        string     `json:"member_id"`
	AccountID       string     `json:"account_id"`
	DisplayName     string     `json:"display_name"`
	Username        string     `json:"username"`
	AvatarObjectKey string     `json:"avatar_object_key"`
	Role            string     `json:"role"`
	LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
	LastReadAt      *time.Time `json:"last_read_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type MessageAggregateSync struct {
	Message   *MessageProjection          `json:"message,omitempty"`
	Members   []RoomMemberProjection      `json:"members,omitempty"`
	Receipts  []MessageReceiptProjection  `json:"receipts,omitempty"`
	Deletions []MessageDeletionProjection `json:"deletions,omitempty"`
}

type MessageProjection struct {
	RoomID                 string              `json:"room_id"`
	RoomName               string              `json:"room_name"`
	RoomType               string              `json:"room_type"`
	MessageID              string              `json:"message_id"`
	MessageContent         string              `json:"message_content"`
	MessageType            string              `json:"message_type"`
	ReplyToMessageID       string              `json:"reply_to_message_id"`
	ForwardedFromMessageID string              `json:"forwarded_from_message_id"`
	FileName               string              `json:"file_name"`
	FileSize               int64               `json:"file_size"`
	MimeType               string              `json:"mime_type"`
	ObjectKey              string              `json:"object_key"`
	MessageSenderID        string              `json:"message_sender_id"`
	MessageSenderName      string              `json:"message_sender_name"`
	MessageSenderEmail     string              `json:"message_sender_email"`
	MessageSentAt          time.Time           `json:"message_sent_at"`
	Mentions               []ProjectionMention `json:"mentions"`
	MentionAll             bool                `json:"mention_all"`
	MentionedAccountIDs    []string            `json:"mentioned_account_ids"`
	EditedAt               *time.Time          `json:"edited_at,omitempty"`
	DeletedForEveryoneAt   *time.Time          `json:"deleted_for_everyone_at,omitempty"`
}

type MessageReceiptProjection struct {
	RoomID      string     `json:"room_id"`
	MessageID   string     `json:"message_id"`
	AccountID   string     `json:"account_id"`
	Status      string     `json:"status"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	SeenAt      *time.Time `json:"seen_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type MessageDeletionProjection struct {
	RoomID        string    `json:"room_id"`
	MessageID     string    `json:"message_id"`
	AccountID     string    `json:"account_id"`
	MessageSentAt time.Time `json:"message_sent_at"`
	CreatedAt     time.Time `json:"created_at"`
}
