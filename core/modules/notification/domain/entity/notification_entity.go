package entity

import (
	"time"

	"wechat-clone/core/modules/notification/types"
)

type NotificationEntity struct {
	ID                 string
	AccountID          string
	Kind               types.NotificationKind
	Type               types.NotificationType
	GroupKey           string
	Subject            string
	Body               string
	IsRead             bool
	ReadAt             *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SortAt             time.Time
	RoomID             string
	RoomName           string
	SenderID           string
	SenderName         string
	MessageCount       int
	LastMessageID      string
	LastMessagePreview string
	LastMessageAt      *time.Time
}
