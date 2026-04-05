// CODE_GENERATOR: request

package in

import "errors"

type GetAccountBalanceRequest struct {
	AccountId string `json:"account_id" form:"account_id" binding:"required"`
}

func (r *GetAccountBalanceRequest) Validate() error {
	if r.AccountId == "" {
		return errors.New("account_id is required")
	}
	return nil
}
