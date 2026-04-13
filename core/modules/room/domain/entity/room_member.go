package entity

import (
	"go-socket/core/modules/room/types"
	"time"
)

type RoomMemberEntity struct {
	ID              string         `json:"id"`
	RoomID          string         `json:"room_id"`
	AccountID       string         `json:"account_id"`
	DisplayName     string         `json:"display_name"`
	Username        string         `json:"username"`
	AvatarObjectKey string         `json:"avatar_object_key"`
	Role            types.RoomRole `json:"role"`
	LastDeliveredAt *time.Time     `json:"last_delivered_at,omitempty"`
	LastReadAt      *time.Time     `json:"last_read_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
