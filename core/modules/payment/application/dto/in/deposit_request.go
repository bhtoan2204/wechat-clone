// CODE_GENERATOR: request

package in

import (
	"errors"
)

type DepositRequest struct {
	Amount int64 `json:"amount" form:"amount" binding:"required"`
}

func (r *DepositRequest) Validate() error {
	if r.Amount == 0 {
		return errors.New("amount is required")
	}
	return nil
}
