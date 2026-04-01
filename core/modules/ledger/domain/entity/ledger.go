package entity

import "time"

type LedgerEntry struct {
	ID            int64     `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Amount        int64     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

type LedgerEntryInput struct {
	AccountID string
	Amount    int64
}

type LedgerTransaction struct {
	TransactionID string
	CreatedAt     time.Time
	Entries       []*LedgerEntry
}
