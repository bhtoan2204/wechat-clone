// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type CreatePaymentRequest struct {
	Provider        string            `json:"provider" form:"provider" binding:"required"`
	Amount          int64             `json:"amount" form:"amount" binding:"required"`
	Currency        string            `json:"currency" form:"currency" binding:"required"`
	CreditAccountID string            `json:"credit_account_id" form:"credit_account_id"`
	Metadata        map[string]string `json:"metadata" form:"metadata"`
}

func (r *CreatePaymentRequest) Normalize() {
	r.Provider = strings.TrimSpace(r.Provider)
	r.Currency = strings.TrimSpace(r.Currency)
	r.CreditAccountID = strings.TrimSpace(r.CreditAccountID)
	for key, value := range r.Metadata {
		r.Metadata[key] = strings.TrimSpace(value)
	}
}

func (r *CreatePaymentRequest) Validate() error {
	r.Normalize()
	if r.Provider == "" {
		return stackErr.Error(errors.New("provider is required"))
	}
	if r.Amount == 0 {
		return stackErr.Error(errors.New("amount is required"))
	}
	if r.Currency == "" {
		return stackErr.Error(errors.New("currency is required"))
	}
	return nil
}
