package events

import "time"

const (
	EventLedgerAccountTransferredToAccount = "EventLedgerAccountTransferredToAccount"
)

type LedgerEntry struct {
	ID            int64     `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Currency      string    `json:"currency"`
	Amount        int64     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

type LedgerTransaction struct {
	TransactionID string
	Currency      string
	CreatedAt     time.Time
	Entries       []*LedgerEntry
}
