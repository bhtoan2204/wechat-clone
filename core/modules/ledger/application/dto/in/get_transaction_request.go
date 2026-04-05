// CODE_GENERATOR: request

package in

import "errors"

type GetTransactionRequest struct {
	TransactionId string `json:"transaction_id" form:"transaction_id" binding:"required"`
}

func (r *GetTransactionRequest) Validate() error {
	if r.TransactionId == "" {
		return errors.New("transaction_id is required")
	}
	return nil
}
