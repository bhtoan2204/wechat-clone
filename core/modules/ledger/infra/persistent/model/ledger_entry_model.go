package model

import "time"

type LedgerEntryModel struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	TransactionID string    `gorm:"not null"`
	AccountID     string    `gorm:"not null"`
	Currency      string    `gorm:"not null"`
	Amount        int64     `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (LedgerEntryModel) TableName() string {
	return "ledger_entries"
}
