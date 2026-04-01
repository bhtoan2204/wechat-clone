package model

import "time"

type LedgerTransactionModel struct {
	TransactionID string    `gorm:"column:transaction_id;primaryKey"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (LedgerTransactionModel) TableName() string {
	return "ledger_transactions"
}
