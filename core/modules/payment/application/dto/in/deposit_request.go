package in

import "errors"

type DepositRequest struct {
	Amount int64 `json:"amount" form:"amount"`
}

func (r *DepositRequest) Validate() error {
	if r.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	return nil
}
