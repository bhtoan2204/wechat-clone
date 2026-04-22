package model

import "time"

type LedgerPostedTransactionModel struct {
	ID            string    `gorm:"primaryKey"`
	AggregateID   string    `gorm:"not null;uniqueIndex:idx_ledger_posted_tx_agg_type_tx"`
	AggregateType string    `gorm:"not null;uniqueIndex:idx_ledger_posted_tx_agg_type_tx"`
	TransactionID string    `gorm:"not null;uniqueIndex:idx_ledger_posted_tx_agg_type_tx"`
	EventName     string    `gorm:"not null"`
	EventData     string    `gorm:"type:text;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (LedgerPostedTransactionModel) TableName() string {
	return "ledger_posted_transactions"
}
