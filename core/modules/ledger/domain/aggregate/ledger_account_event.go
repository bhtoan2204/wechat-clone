package aggregate

import (
	"time"

	"wechat-clone/core/shared/pkg/event"
)

var (
	EventNameLedgerAccountDepositFromIntent      = event.EventName((*EventLedgerAccountDepositFromIntent)(nil))
	EventNameLedgerAccountWithdrawFromIntent     = event.EventName((*EventLedgerAccountWithdrawFromIntent)(nil))
	EventNameLedgerAccountDepositFromRefund      = event.EventName((*EventLedgerAccountDepositFromRefund)(nil))
	EventNameLedgerAccountWithdrawFromRefund     = event.EventName((*EventLedgerAccountWithdrawFromRefund)(nil))
	EventNameLedgerAccountDepositFromChargeback  = event.EventName((*EventLedgerAccountDepositFromChargeback)(nil))
	EventNameLedgerAccountWithdrawFromChargeback = event.EventName((*EventLedgerAccountWithdrawFromChargeback)(nil))
	EventNameLedgerAccountTransferredToAccount   = event.EventName((*EventLedgerAccountTransferredToAccount)(nil))
	EventNameLedgerAccountReceivedTransfer       = event.EventName((*EventLedgerAccountReceivedTransfer)(nil))
)

type EventLedgerAccountDepositFromIntent struct {
	TransactionID         string    `json:"transaction_id"`
	PaymentID             string    `json:"payment_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	Amount                int64     `json:"amount"`
	BookedAt              time.Time `json:"booked_at"`
}

type EventLedgerAccountWithdrawFromIntent struct {
	TransactionID         string    `json:"transaction_id"`
	PaymentID             string    `json:"payment_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	Amount                int64     `json:"amount"`
	BookedAt              time.Time `json:"booked_at"`
}

type EventLedgerAccountDepositFromRefund struct {
	TransactionID         string    `json:"transaction_id"`
	PaymentID             string    `json:"payment_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	Amount                int64     `json:"amount"`
	BookedAt              time.Time `json:"booked_at"`
}

type EventLedgerAccountWithdrawFromRefund struct {
	TransactionID         string    `json:"transaction_id"`
	PaymentID             string    `json:"payment_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	Amount                int64     `json:"amount"`
	BookedAt              time.Time `json:"booked_at"`
}

type EventLedgerAccountDepositFromChargeback struct {
	TransactionID         string    `json:"transaction_id"`
	PaymentID             string    `json:"payment_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	Amount                int64     `json:"amount"`
	BookedAt              time.Time `json:"booked_at"`
}

type EventLedgerAccountWithdrawFromChargeback struct {
	TransactionID         string    `json:"transaction_id"`
	PaymentID             string    `json:"payment_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	Amount                int64     `json:"amount"`
	BookedAt              time.Time `json:"booked_at"`
}

func (e *EventLedgerAccountDepositFromIntent) paymentEvent() ledgerPaymentEventData {
	return ledgerPaymentEventData{
		transactionID:         e.TransactionID,
		paymentID:             e.PaymentID,
		counterpartyAccountID: e.CounterpartyAccountID,
		currency:              e.Currency,
		amount:                e.Amount,
		bookedAt:              e.BookedAt,
	}
}

func (e *EventLedgerAccountWithdrawFromIntent) paymentEvent() ledgerPaymentEventData {
	return ledgerPaymentEventData{
		transactionID:         e.TransactionID,
		paymentID:             e.PaymentID,
		counterpartyAccountID: e.CounterpartyAccountID,
		currency:              e.Currency,
		amount:                e.Amount,
		bookedAt:              e.BookedAt,
	}
}

func (e *EventLedgerAccountDepositFromRefund) paymentEvent() ledgerPaymentEventData {
	return ledgerPaymentEventData{
		transactionID:         e.TransactionID,
		paymentID:             e.PaymentID,
		counterpartyAccountID: e.CounterpartyAccountID,
		currency:              e.Currency,
		amount:                e.Amount,
		bookedAt:              e.BookedAt,
	}
}

func (e *EventLedgerAccountWithdrawFromRefund) paymentEvent() ledgerPaymentEventData {
	return ledgerPaymentEventData{
		transactionID:         e.TransactionID,
		paymentID:             e.PaymentID,
		counterpartyAccountID: e.CounterpartyAccountID,
		currency:              e.Currency,
		amount:                e.Amount,
		bookedAt:              e.BookedAt,
	}
}

func (e *EventLedgerAccountDepositFromChargeback) paymentEvent() ledgerPaymentEventData {
	return ledgerPaymentEventData{
		transactionID:         e.TransactionID,
		paymentID:             e.PaymentID,
		counterpartyAccountID: e.CounterpartyAccountID,
		currency:              e.Currency,
		amount:                e.Amount,
		bookedAt:              e.BookedAt,
	}
}

func (e *EventLedgerAccountWithdrawFromChargeback) paymentEvent() ledgerPaymentEventData {
	return ledgerPaymentEventData{
		transactionID:         e.TransactionID,
		paymentID:             e.PaymentID,
		counterpartyAccountID: e.CounterpartyAccountID,
		currency:              e.Currency,
		amount:                e.Amount,
		bookedAt:              e.BookedAt,
	}
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
