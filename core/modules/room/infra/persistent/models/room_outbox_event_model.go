package models

import "time"

type RoomOutboxEventModel struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	AggregateID   string    `gorm:"not null;index"`
	AggregateType string    `gorm:"not null;index"`
	Version       int       `gorm:"not null"`
	EventName     string    `gorm:"not null;index"`
	EventData     string    `gorm:"type:CLOB;not null"`
	Metadata      string    `gorm:"type:CLOB;not null;default:'{}'"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (RoomOutboxEventModel) TableName() string {
	return "room_outbox_events"
}
