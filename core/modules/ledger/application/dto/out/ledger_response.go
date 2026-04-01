package out

import "time"

type LedgerEntryResponse struct {
	ID            int64     `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Amount        int64     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

type TransactionResponse struct {
	TransactionID string                `json:"transaction_id"`
	CreatedAt     time.Time             `json:"created_at"`
	Entries       []LedgerEntryResponse `json:"entries"`
}

type AccountBalanceResponse struct {
	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`
}
