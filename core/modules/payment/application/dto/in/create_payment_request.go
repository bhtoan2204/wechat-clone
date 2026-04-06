// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type CreatePaymentRequest struct {
	Provider        string            `json:"provider" form:"provider" binding:"required"`
	TransactionID   string            `json:"transaction_id" form:"transaction_id" binding:"required"`
	Amount          int64             `json:"amount" form:"amount" binding:"required"`
	Currency        string            `json:"currency" form:"currency" binding:"required"`
	DebitAccountID  string            `json:"debit_account_id" form:"debit_account_id" binding:"required"`
	CreditAccountID string            `json:"credit_account_id" form:"credit_account_id" binding:"required"`
	Metadata        map[string]string `json:"metadata" form:"metadata"`
}

func (r *CreatePaymentRequest) Normalize() {
	r.Provider = strings.TrimSpace(r.Provider)
	r.TransactionID = strings.TrimSpace(r.TransactionID)
	r.Currency = strings.TrimSpace(r.Currency)
	r.DebitAccountID = strings.TrimSpace(r.DebitAccountID)
	r.CreditAccountID = strings.TrimSpace(r.CreditAccountID)
	for key, value := range r.Metadata {
		r.Metadata[key] = strings.TrimSpace(value)
	}
}

func (r *CreatePaymentRequest) Validate() error {
	r.Normalize()
	if r.Provider == "" {
		return errors.New("provider is required")
	}
	if r.TransactionID == "" {
		return errors.New("transaction_id is required")
	}
	if r.Amount == 0 {
		return errors.New("amount is required")
	}
	if r.Currency == "" {
		return errors.New("currency is required")
	}
	if r.DebitAccountID == "" {
		return errors.New("debit_account_id is required")
	}
	if r.CreditAccountID == "" {
		return errors.New("credit_account_id is required")
	}
	return nil
}
