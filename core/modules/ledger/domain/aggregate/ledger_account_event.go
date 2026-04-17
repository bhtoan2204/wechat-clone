package aggregate

import (
	"reflect"
	"time"
)

var (
	EventNameLedgerAccountPaymentBooked        = eventName((*EventLedgerAccountPaymentBooked)(nil))
	EventNameLedgerAccountTransferredToAccount = eventName((*EventLedgerAccountTransferredToAccount)(nil))
	EventNameLedgerAccountReceivedTransfer     = eventName((*EventLedgerAccountReceivedTransfer)(nil))
)

type EventLedgerAccountPaymentBooked struct {
	TransactionID         string    `json:"transaction_id"`
	ReferenceType         string    `json:"reference_type,omitempty"`
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

func eventName(payload any) string {
	typ := reflect.TypeOf(payload)
	if typ == nil {
		return ""
	}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}
