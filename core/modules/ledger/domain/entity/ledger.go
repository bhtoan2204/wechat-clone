package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrLedgerTransactionIDRequired = errors.New("transaction_id is required")
	ErrLedgerEntriesRequired       = errors.New("at least 2 entries are required")
	ErrLedgerEntriesUnbalanced     = errors.New("entries must balance to zero")
	ErrLedgerEntryAmountZero       = errors.New("amount must be non-zero")
	ErrLedgerEntryAccountRequired  = errors.New("account_id is required")
)

type LedgerEntry struct {
	ID            int64     `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Amount        int64     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

type LedgerEntryInput struct {
	AccountID string
	Amount    int64
}

type LedgerTransaction struct {
	TransactionID string
	CreatedAt     time.Time
	Entries       []*LedgerEntry
}

func NewLedgerTransaction(transactionID string, entries []LedgerEntryInput, now time.Time) (*LedgerTransaction, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, ErrLedgerTransactionIDRequired
	}
	if len(entries) < 2 {
		return nil, ErrLedgerEntriesRequired
	}

	now = normalizeLedgerTime(now)
	transaction := &LedgerTransaction{
		TransactionID: transactionID,
		CreatedAt:     now,
		Entries:       make([]*LedgerEntry, 0, len(entries)),
	}

	var sum int64
	for idx, entry := range entries {
		accountID := strings.TrimSpace(entry.AccountID)
		if accountID == "" {
			return nil, fmt.Errorf("entries[%d].%v", idx, ErrLedgerEntryAccountRequired)
		}
		if entry.Amount == 0 {
			return nil, fmt.Errorf("entries[%d].%v", idx, ErrLedgerEntryAmountZero)
		}

		sum += entry.Amount
		transaction.Entries = append(transaction.Entries, &LedgerEntry{
			TransactionID: transactionID,
			AccountID:     accountID,
			Amount:        entry.Amount,
			CreatedAt:     now,
		})
	}

	if sum != 0 {
		return nil, ErrLedgerEntriesUnbalanced
	}

	return transaction, nil
}

func normalizeLedgerTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
