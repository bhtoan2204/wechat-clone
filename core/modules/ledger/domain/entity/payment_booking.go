package entity

import (
	"errors"
	"fmt"
	"strings"

	sharedevents "wechat-clone/core/shared/contracts/events"
)

var (
	ErrPaymentBookingIDRequired            = errors.New("payment_id is required")
	ErrPaymentBookingClearingKeyRequired   = errors.New("clearing_account_key is required")
	ErrPaymentBookingCreditAccountRequired = errors.New("credit_account_id is required")
	ErrPaymentBookingCurrencyRequired      = errors.New("currency is required")
	ErrPaymentBookingAccountsMustDiffer    = errors.New("debit_account_id and credit_account_id must be different")
	ErrPaymentBookingAmountInvalid         = errors.New("amount must be positive")
	ErrPaymentBookingTypeInvalid           = errors.New("payment booking type is invalid")
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

type PaymentReversalBooking struct {
	PaymentID          string
	ClearingAccountKey string
	ReversalType       string
	DebitAccountID     string
	CreditAccountID    string
	Currency           string
	Amount             int64
}

type PaymentReversalBookingInput struct {
	PaymentID          string
	TransactionID      string
	ClearingAccountKey string
	CreditAccountID    string
	Currency           string
	Amount             int64
	ReversalType       string
}

func NewPaymentSucceededBooking(input PaymentSucceededBookingInput) (*PaymentSucceededBooking, error) {
	paymentID := strings.TrimSpace(input.PaymentID)
	if paymentID == "" {
		paymentID = strings.TrimSpace(input.TransactionID)
	}
	if paymentID == "" {
		return nil, ErrPaymentBookingIDRequired
	}

	clearingAccountKey := strings.ToLower(strings.TrimSpace((input.ClearingAccountKey)))
	if clearingAccountKey == "" {
		return nil, ErrPaymentBookingClearingKeyRequired
	}
	creditAccountID := strings.TrimSpace(input.CreditAccountID)
	if creditAccountID == "" {
		return nil, ErrPaymentBookingCreditAccountRequired
	}
	currency := strings.ToUpper(strings.TrimSpace((input.Currency)))
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

func NewPaymentReversalBooking(input PaymentReversalBookingInput) (*PaymentReversalBooking, error) {
	paymentID := strings.TrimSpace(input.PaymentID)
	if paymentID == "" {
		paymentID = strings.TrimSpace(input.TransactionID)
	}
	if paymentID == "" {
		return nil, ErrPaymentBookingIDRequired
	}

	reversalType := normalizePaymentBookingType(input.ReversalType)
	if reversalType == "" {
		return nil, ErrPaymentBookingTypeInvalid
	}

	clearingAccountKey := strings.ToLower(strings.TrimSpace((input.ClearingAccountKey)))
	if clearingAccountKey == "" {
		return nil, ErrPaymentBookingClearingKeyRequired
	}
	debitAccountID := strings.TrimSpace(input.CreditAccountID)
	if debitAccountID == "" {
		return nil, ErrPaymentBookingCreditAccountRequired
	}
	currency := strings.ToUpper(strings.TrimSpace((input.Currency)))
	if currency == "" {
		return nil, ErrPaymentBookingCurrencyRequired
	}
	if input.Amount <= 0 {
		return nil, ErrPaymentBookingAmountInvalid
	}
	creditAccountID := ledgerClearingAccountID(clearingAccountKey)
	if debitAccountID == creditAccountID {
		return nil, ErrPaymentBookingAccountsMustDiffer
	}

	return &PaymentReversalBooking{
		PaymentID:          paymentID,
		ClearingAccountKey: clearingAccountKey,
		ReversalType:       reversalType,
		DebitAccountID:     debitAccountID,
		CreditAccountID:    creditAccountID,
		Currency:           currency,
		Amount:             input.Amount,
	}, nil
}

func (b *PaymentReversalBooking) LedgerTransactionID() string {
	suffix := "reversed"
	switch b.ReversalType {
	case sharedevents.EventPaymentRefunded:
		suffix = "refunded"
	case sharedevents.EventPaymentChargeback:
		suffix = "chargeback"
	}
	return fmt.Sprintf("payment:%s:%s", strings.TrimSpace(b.PaymentID), suffix)
}

func (b *PaymentReversalBooking) LedgerEntries() []LedgerEntryInput {
	return []LedgerEntryInput{
		{AccountID: b.DebitAccountID, Currency: b.Currency, Amount: -b.Amount},
		{AccountID: b.CreditAccountID, Currency: b.Currency, Amount: b.Amount},
	}
}

func ledgerClearingAccountID(clearingAccountKey string) string {
	return fmt.Sprintf("ledger:clearing:%s", strings.ToLower(strings.TrimSpace((clearingAccountKey))))
}

func normalizePaymentBookingType(value string) string {
	switch strings.TrimSpace(value) {
	case sharedevents.EventPaymentRefunded:
		return sharedevents.EventPaymentRefunded
	case sharedevents.EventPaymentChargeback:
		return sharedevents.EventPaymentChargeback
	default:
		return ""
	}
}
