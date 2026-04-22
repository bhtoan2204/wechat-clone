package valueobject

import "time"

type LedgerAccountTransferPostingInput struct {
	AccountID             string
	TransactionID         string
	CounterpartyAccountID string
	Currency              string
	Amount                int64
	BookedAt              time.Time
}
