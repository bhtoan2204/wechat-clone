package events

import (
	"context"
	"time"
)

const (
	EventRoomMessageCreated = "EventRoomMessageCreated"
)

type RoomMessageMention struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name,omitempty"`
	Username    string `json:"username,omitempty"`
}

type RoomMessageCreatedEvent struct {
	RoomID                 string               `json:"room_id"`
	RoomName               string               `json:"room_name,omitempty"`
	RoomType               string               `json:"room_type,omitempty"`
	MessageID              string               `json:"message_id"`
	MessageContent         string               `json:"message_content,omitempty"`
	MessageType            string               `json:"message_type,omitempty"`
	ReplyToMessageID       string               `json:"reply_to_message_id,omitempty"`
	ForwardedFromMessageID string               `json:"forwarded_from_message_id,omitempty"`
	FileName               string               `json:"file_name,omitempty"`
	FileSize               int64                `json:"file_size,omitempty"`
	MimeType               string               `json:"mime_type,omitempty"`
	ObjectKey              string               `json:"object_key,omitempty"`
	MessageSenderID        string               `json:"message_sender_id"`
	MessageSenderName      string               `json:"message_sender_name,omitempty"`
	MessageSenderEmail     string               `json:"message_sender_email,omitempty"`
	MessageSentAt          time.Time            `json:"message_sent_at"`
	Mentions               []RoomMessageMention `json:"mentions,omitempty"`
	MentionAll             bool                 `json:"mention_all"`
	MentionedAccountIDs    []string             `json:"mentioned_account_ids,omitempty"`
}

type TimelineProjector interface {
	ProjectRoom(ctx context.Context, projection *RoomProjection) error
	DeleteProjectedRoom(ctx context.Context, roomID string) error
	ProjectRoomMember(ctx context.Context, projection *RoomMemberProjection) error
	DeleteProjectedRoomMember(ctx context.Context, roomID, accountID string) error
	ProjectMessage(ctx context.Context, projection *TimelineMessageProjection) error
	ProjectMessageReceipt(ctx context.Context, projection *MessageReceiptProjection) error
	ProjectMessageDeletion(ctx context.Context, projection *MessageDeletionProjection) error
}

type MessageSearchIndexer interface {
	UpsertMessage(ctx context.Context, document *SearchMessageDocument) error
}

type ProjectionMention struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}

type TimelineMessageProjection struct {
	RoomName               string              `json:"room_name"`
	RoomType               string              `json:"room_type"`
	RoomID                 string              `json:"room_id"`
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

type SearchMessageDocument struct {
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
}

type RoomProjection struct {
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

type RoomMemberProjection struct {
	RoomID          string     `json:"room_id"`
	MemberID        string     `json:"member_id"`
	AccountID       string     `json:"account_id"`
	Role            string     `json:"role"`
	LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
	LastReadAt      *time.Time `json:"last_read_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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
