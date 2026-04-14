// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
)

type WithdrawalRequest struct {
	Amount int64 `json:"amount" form:"amount" binding:"required"`
}

func (r *WithdrawalRequest) Validate() error {
	if r.Amount == 0 {
		return stackErr.Error(errors.New("amount is required"))
	}
	return nil
}
