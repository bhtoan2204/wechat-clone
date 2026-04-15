package model

import "time"

type LedgerOutboxEventModel struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	AggregateID   string    `gorm:"not null;index"`
	AggregateType string    `gorm:"not null;index"`
	Version       int       `gorm:"not null"`
	EventName     string    `gorm:"not null;index"`
	EventData     string    `gorm:"type:CLOB;not null"`
	Metadata      string    `gorm:"type:CLOB;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (LedgerOutboxEventModel) TableName() string {
	return "ledger_outbox_events"
}
