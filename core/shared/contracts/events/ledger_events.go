package events

import "time"

const (
	EventLedgerAccountTransferredToAccount = "EventLedgerAccountTransferredToAccount"
)

type LedgerAccountTransferredToAccountEvent struct {
	TransactionID string    `json:"transaction_id"`
	ToAccountID   string    `json:"to_account_id"`
	Currency      string    `json:"currency"`
	Amount        int64     `json:"amount"`
	BookedAt      time.Time `json:"booked_at"`
}

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
