// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type TransferRequest struct {
	Amount     int64  `json:"amount" form:"amount" binding:"required"`
	ReceiverID string `json:"receiver_id" form:"receiver_id" binding:"required"`
}

func (r *TransferRequest) Normalize() {
	r.ReceiverID = strings.TrimSpace(r.ReceiverID)
}

func (r *TransferRequest) Validate() error {
	r.Normalize()
	if r.Amount == 0 {
		return errors.New("amount is required")
	}
	if r.ReceiverID == "" {
		return errors.New("receiver_id is required")
	}
	return nil
}
