// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
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

func (r *LedgerEntryRequest) Normalize() {
	r.AccountID = strings.TrimSpace(r.AccountID)
}

func (r *CreateTransactionRequest) Normalize() {
	r.TransactionID = strings.TrimSpace(r.TransactionID)
	for idx := range r.Entries {
		r.Entries[idx].Normalize()
	}
}

func (r *CreateTransactionRequest) Validate() error {
	r.Normalize()
	if r.TransactionID == "" {
		return stackErr.Error(errors.New("transaction_id is required"))
	}
	if len(r.Entries) == 0 {
		return stackErr.Error(errors.New("entries is required"))
	}
	return nil
}
