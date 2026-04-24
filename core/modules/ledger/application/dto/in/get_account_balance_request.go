// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"strings"
	"wechat-clone/core/shared/pkg/stackErr"
)

type GetAccountBalanceRequest struct {
	Currency string `json:"currency" form:"currency" binding:"required"`
}

func (r *GetAccountBalanceRequest) Normalize() {
	r.Currency = strings.TrimSpace(r.Currency)
}

func (r *GetAccountBalanceRequest) Validate() error {
	r.Normalize()
	if r.Currency == "" {
		return stackErr.Error(errors.New("currency is required"))
	}
	return nil
}
