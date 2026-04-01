package in

import (
	"errors"
	"fmt"
	"strings"
)

type CreateTransactionRequest struct {
	TransactionID string             `json:"transaction_id"`
	Entries       []LedgerEntryInput `json:"entries"`
}

type LedgerEntryInput struct {
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount"`
}

func (r *CreateTransactionRequest) Validate() error {
	r.TransactionID = strings.TrimSpace(r.TransactionID)
	if r.TransactionID == "" {
		return errors.New("transaction_id is required")
	}
	if len(r.Entries) < 2 {
		return errors.New("at least 2 entries are required")
	}

	var sum int64
	for idx, entry := range r.Entries {
		if strings.TrimSpace(entry.AccountID) == "" {
			return fmt.Errorf("entries[%d].account_id is required", idx)
		}
		if entry.Amount == 0 {
			return fmt.Errorf("entries[%d].amount must be non-zero", idx)
		}
		sum += entry.Amount
	}

	if sum != 0 {
		return errors.New("sum(entries.amount) must equal 0")
	}

	return nil
}
