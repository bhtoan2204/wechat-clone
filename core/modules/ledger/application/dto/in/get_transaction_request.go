// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type GetTransactionRequest struct {
	TransactionID string `json:"transaction_id" form:"transaction_id" binding:"required"`
}

func (r *GetTransactionRequest) Normalize() {
	r.TransactionID = strings.TrimSpace(r.TransactionID)
}

func (r *GetTransactionRequest) Validate() error {
	r.Normalize()
	if r.TransactionID == "" {
		return stackErr.Error(errors.New("transaction_id is required"))
	}
	return nil
}
