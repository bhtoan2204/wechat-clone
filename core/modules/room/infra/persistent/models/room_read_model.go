package models

import (
	"go-socket/core/modules/room/types"
	"time"
)

type RoomReadModel struct {
	ID                  string         `gorm:"primaryKey"`
	Name                string         `gorm:"not null"`
	Description         string         `gorm:"default:''"`
	RoomType            types.RoomType `gorm:"not null"`
	OwnerID             string         `gorm:"not null"`
	DirectKey           *string        `gorm:"index"`
	PinnedMessageID     *string
	MemberCount         int     `gorm:"not null;default:0"`
	LastMessageID       *string `gorm:"index"`
	LastMessageAt       *time.Time
	LastMessageContent  *string
	LastMessageSenderID *string
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
}

func (RoomReadModel) TableName() string {
	return "room_read_models"
}
