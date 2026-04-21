package views

import "time"

type LedgerTransactionEntry struct {
	AccountID string    `json:"account_id"`
	Currency  string    `json:"currency"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type LedgerTransactionProjected struct {
	TransactionID string                   `json:"transaction_id"`
	ReferenceType string                   `json:"reference_type"`
	ReferenceID   string                   `json:"reference_id"`
	Currency      string                   `json:"currency"`
	CreatedAt     time.Time                `json:"created_at"`
	Entries       []LedgerTransactionEntry `json:"entries"`
}

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
