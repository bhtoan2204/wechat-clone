package model

import "time"

type LedgerEntryModel struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	TransactionID string    `gorm:"column:transaction_id;not null"`
	AccountID     string    `gorm:"column:account_id;not null"`
	Amount        int64     `gorm:"column:amount;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (LedgerEntryModel) TableName() string {
	return "ledger_entries"
}
