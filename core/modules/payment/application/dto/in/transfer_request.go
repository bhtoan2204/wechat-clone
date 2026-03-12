package in

import "errors"

type TransferRequest struct {
	Amount     int64  `json:"amount" form:"amount"`
	ReceiverID string `json:"receiver_id" form:"receiver_id"`
}

func (r *TransferRequest) Validate() error {
	if r.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if r.ReceiverID == "" {
		return errors.New("receiver_id is required")
	}
	return nil
}
