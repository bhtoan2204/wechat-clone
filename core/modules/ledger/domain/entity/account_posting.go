package entity

import "time"

const LedgerReferenceInternalTransfer = "ledger.transfer.internal"

type LedgerAccountPosting struct {
	TransactionID         string    `json:"transaction_id"`
	ReferenceType         string    `json:"reference_type"`
	ReferenceID           string    `json:"reference_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	AmountDelta           int64     `json:"amount_delta"`
	BookedAt              time.Time `json:"booked_at"`
}
