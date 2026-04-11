package in

import (
	"errors"
	"strings"

	"go-socket/core/shared/pkg/stackErr"
)

type CreatePaymentRequest struct {
	Provider        string            `json:"provider"`
	TransactionID   string            `json:"transaction_id"`
	Amount          int64             `json:"amount"`
	Currency        string            `json:"currency"`
	DebitAccountID  string            `json:"debit_account_id,omitempty"`
	CreditAccountID string            `json:"credit_account_id"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

func (r *CreatePaymentRequest) Normalize() {
	r.Provider = strings.ToLower(strings.TrimSpace(r.Provider))
	r.TransactionID = strings.TrimSpace(r.TransactionID)
	r.Currency = strings.ToUpper(strings.TrimSpace(r.Currency))
	r.DebitAccountID = strings.TrimSpace(r.DebitAccountID)
	r.CreditAccountID = strings.TrimSpace(r.CreditAccountID)
}

func (r *CreatePaymentRequest) Validate() error {
	r.Normalize()
	if r.Provider == "" {
		return stackErr.Error(errors.New("provider is required"))
	}
	if r.TransactionID == "" {
		return stackErr.Error(errors.New("transaction_id is required"))
	}
	if r.Amount <= 0 {
		return stackErr.Error(errors.New("amount must be greater than 0"))
	}
	if r.Currency == "" {
		return stackErr.Error(errors.New("currency is required"))
	}
	if r.DebitAccountID == "" {
		return stackErr.Error(errors.New("debit_account_id is required"))
	}
	if r.CreditAccountID == "" {
		return stackErr.Error(errors.New("credit_account_id is required"))
	}
	if r.DebitAccountID == r.CreditAccountID {
		return stackErr.Error(errors.New("debit_account_id and credit_account_id must be different"))
	}
	return nil
}
