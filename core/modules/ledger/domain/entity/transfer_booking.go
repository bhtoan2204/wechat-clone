package entity

import (
	"errors"
	"strings"
)

var (
	ErrTransferFromAccountRequired = errors.New("from_account_id is required")
	ErrTransferToAccountRequired   = errors.New("to_account_id is required")
	ErrTransferAccountsMustDiffer  = errors.New("from_account_id and to_account_id must be different")
	ErrTransferCurrencyRequired    = errors.New("currency is required")
	ErrTransferAmountInvalid       = errors.New("amount must be greater than 0")
)

type TransferBooking struct {
	FromAccountID string
	ToAccountID   string
	Currency      string
	Amount        int64
}

type TransferBookingInput struct {
	FromAccountID string
	ToAccountID   string
	Currency      string
	Amount        int64
}

func NewTransferBooking(input TransferBookingInput) (*TransferBooking, error) {
	fromAccountID := strings.TrimSpace(input.FromAccountID)
	if fromAccountID == "" {
		return nil, ErrTransferFromAccountRequired
	}

	toAccountID := strings.TrimSpace(input.ToAccountID)
	if toAccountID == "" {
		return nil, ErrTransferToAccountRequired
	}
	if fromAccountID == toAccountID {
		return nil, ErrTransferAccountsMustDiffer
	}

	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		return nil, ErrTransferCurrencyRequired
	}
	if input.Amount <= 0 {
		return nil, ErrTransferAmountInvalid
	}

	return &TransferBooking{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Currency:      currency,
		Amount:        input.Amount,
	}, nil
}

func (b *TransferBooking) LedgerEntries() []LedgerEntryInput {
	return []LedgerEntryInput{
		{
			AccountID: b.FromAccountID,
			Currency:  b.Currency,
			Amount:    -b.Amount,
		},
		{
			AccountID: b.ToAccountID,
			Currency:  b.Currency,
			Amount:    b.Amount,
		},
	}
}
