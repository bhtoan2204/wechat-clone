// CODE_GENERATOR: request

package in

import (
	"errors"
)

type WithdrawalRequest struct {
	Amount int64 `json:"amount" form:"amount" binding:"required"`
}

func (r *WithdrawalRequest) Validate() error {
	if r.Amount == 0 {
		return errors.New("amount is required")
	}
	return nil
}
