package projection

import "time"

const (
	AggregateTypeLedgerTransactionProjection = "LedgerTransactionProjection"
	EventLedgerTransactionProjected          = "ledger.transaction.projected"
)

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
