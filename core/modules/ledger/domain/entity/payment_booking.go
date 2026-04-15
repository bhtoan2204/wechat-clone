package entity

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrPaymentBookingIDRequired            = errors.New("payment_id is required")
	ErrPaymentBookingClearingKeyRequired   = errors.New("clearing_account_key is required")
	ErrPaymentBookingCreditAccountRequired = errors.New("credit_account_id is required")
	ErrPaymentBookingCurrencyRequired      = errors.New("currency is required")
	ErrPaymentBookingAccountsMustDiffer    = errors.New("debit_account_id and credit_account_id must be different")
	ErrPaymentBookingAmountInvalid         = errors.New("amount must be positive")
)

type PaymentSucceededBooking struct {
	PaymentID          string
	ClearingAccountKey string
	DebitAccountID     string
	CreditAccountID    string
	Currency           string
	Amount             int64
}

type PaymentSucceededBookingInput struct {
	PaymentID          string
	TransactionID      string
	ClearingAccountKey string
	CreditAccountID    string
	Currency           string
	Amount             int64
}

func NewPaymentSucceededBooking(input PaymentSucceededBookingInput) (*PaymentSucceededBooking, error) {
	paymentID := strings.TrimSpace(input.PaymentID)
	if paymentID == "" {
		paymentID = strings.TrimSpace(input.TransactionID)
	}
	if paymentID == "" {
		return nil, ErrPaymentBookingIDRequired
	}

	clearingAccountKey := normalizePaymentClearingAccountKey(input.ClearingAccountKey)
	if clearingAccountKey == "" {
		return nil, ErrPaymentBookingClearingKeyRequired
	}
	creditAccountID := strings.TrimSpace(input.CreditAccountID)
	if creditAccountID == "" {
		return nil, ErrPaymentBookingCreditAccountRequired
	}
	currency := normalizeLedgerCurrency(input.Currency)
	if currency == "" {
		return nil, ErrPaymentBookingCurrencyRequired
	}
	if input.Amount <= 0 {
		return nil, ErrPaymentBookingAmountInvalid
	}
	debitAccountID := ledgerClearingAccountID(clearingAccountKey)
	if debitAccountID == creditAccountID {
		return nil, ErrPaymentBookingAccountsMustDiffer
	}

	return &PaymentSucceededBooking{
		PaymentID:          paymentID,
		ClearingAccountKey: clearingAccountKey,
		DebitAccountID:     debitAccountID,
		CreditAccountID:    creditAccountID,
		Currency:           currency,
		Amount:             input.Amount,
	}, nil
}

func (b *PaymentSucceededBooking) LedgerTransactionID() string {
	return fmt.Sprintf("payment:%s:succeeded", strings.TrimSpace(b.PaymentID))
}

func (b *PaymentSucceededBooking) LedgerEntries() []LedgerEntryInput {
	return []LedgerEntryInput{
		{AccountID: b.DebitAccountID, Currency: b.Currency, Amount: -b.Amount},
		{AccountID: b.CreditAccountID, Currency: b.Currency, Amount: b.Amount},
	}
}

func normalizePaymentClearingAccountKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func ledgerClearingAccountID(clearingAccountKey string) string {
	return fmt.Sprintf("ledger:clearing:%s", normalizePaymentClearingAccountKey(clearingAccountKey))
}
