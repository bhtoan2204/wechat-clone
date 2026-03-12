package model

import "time"

type PaymentOutboxEventModel struct {
	ID            int64     `gorm:"primaryKey"`
	AggregateID   string    `gorm:"index"`
	AggregateType string    `gorm:"index"`
	Version       int       `gorm:"not null"`
	EventName     string    `gorm:"not null"`
	EventData     string    `gorm:"type:JSON;not null"`
	Metadata      string    `gorm:"type:JSON;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (PaymentOutboxEventModel) TableName() string {
	return "payment_outbox_events"
}
