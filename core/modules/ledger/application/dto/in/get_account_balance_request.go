// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type GetAccountBalanceRequest struct {
	AccountID string `json:"account_id" form:"account_id" binding:"required"`
}

func (r *GetAccountBalanceRequest) Normalize() {
	r.AccountID = strings.TrimSpace(r.AccountID)
}

func (r *GetAccountBalanceRequest) Validate() error {
	r.Normalize()
	if r.AccountID == "" {
		return stackErr.Error(errors.New("account_id is required"))
	}
	return nil
}
