package views

import "time"

type LedgerTransactionListRow struct {
	TransactionID string
	Currency      string
	CreatedAt     time.Time
}

type LedgerTransactionModel struct {
	TransactionID string    `gorm:"primaryKey"`
	Currency      string    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (LedgerTransactionModel) TableName() string {
	return "ledger_transactions"
}
