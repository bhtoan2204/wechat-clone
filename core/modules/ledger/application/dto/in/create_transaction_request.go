// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type CreateTransactionRequest struct {
	TransactionID string               `json:"transaction_id" form:"transaction_id" binding:"required"`
	Entries       []LedgerEntryRequest `json:"entries" form:"entries" binding:"required"`
}

type LedgerEntryRequest struct {
	AccountID string `json:"account_id" form:"account_id" binding:"required"`
	Amount    int64  `json:"amount" form:"amount" binding:"required"`
}

func (r *CreateTransactionRequest) Normalize() {
	r.TransactionID = strings.TrimSpace(r.TransactionID)
}

func (r *CreateTransactionRequest) Validate() error {
	r.Normalize()
	if r.TransactionID == "" {
		return errors.New("transaction_id is required")
	}
	if len(r.Entries) == 0 {
		return errors.New("entries is required")
	}
	return nil
}
