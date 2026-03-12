package in

import "errors"

type WithdrawalRequest struct {
	Amount int64 `json:"amount" form:"amount"`
}

func (r *WithdrawalRequest) Validate() error {
	if r.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	return nil
}
