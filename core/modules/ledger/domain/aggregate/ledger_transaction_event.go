package aggregate

import "time"

type LedgerTransactionEntryPayload struct {
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount"`
}

type EventLedgerTransactionCreated struct {
	TransactionID string                          `json:"transaction_id"`
	Currency      string                          `json:"currency"`
	CreatedAt     time.Time                       `json:"created_at"`
	Entries       []LedgerTransactionEntryPayload `json:"entries"`
}
