package aggregate

import "time"

type EventLedgerAccountPaymentBooked struct {
	TransactionID         string    `json:"transaction_id"`
	PaymentID             string    `json:"payment_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	AmountDelta           int64     `json:"amount_delta"`
	BookedAt              time.Time `json:"booked_at"`
}

type EventLedgerAccountTransferredToAccount struct {
	TransactionID string    `json:"transaction_id"`
	ToAccountID   string    `json:"to_account_id"`
	Currency      string    `json:"currency"`
	Amount        int64     `json:"amount"`
	BookedAt      time.Time `json:"booked_at"`
}

type EventLedgerAccountReceivedTransfer struct {
	TransactionID string    `json:"transaction_id"`
	FromAccountID string    `json:"from_account_id"`
	Currency      string    `json:"currency"`
	Amount        int64     `json:"amount"`
	BookedAt      time.Time `json:"booked_at"`
}
