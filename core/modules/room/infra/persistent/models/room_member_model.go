package models

import (
	"go-socket/core/modules/room/types"
	"time"
)

type RoomMemberModel struct {
	ID              string         `gorm:"primaryKey"`
	RoomID          string         `gorm:"not null;index"`
	AccountID       string         `gorm:"not null;index"`
	Role            types.RoomRole `gorm:"default:member"`
	LastDeliveredAt *time.Time
	LastReadAt      *time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (RoomMemberModel) TableName() string {
	return "room_members"
}
