package valueobject

import "time"

type LedgerAccountPostingInput struct {
	AccountID             string
	TransactionID         string
	ReferenceType         string
	ReferenceID           string
	CounterpartyAccountID string
	Currency              string
	AmountDelta           int64
	BookedAt              time.Time
}
