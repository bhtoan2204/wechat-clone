// CODE_GENERATOR: request

package in

import "errors"

type ProcessWebhookRequest struct {
	Provider  string `json:"provider" form:"provider" binding:"required"`
	Signature string `json:"signature" form:"signature"`
	Payload   string `json:"payload" form:"payload"`
}

func (r *ProcessWebhookRequest) Validate() error {
	if r.Provider == "" {
		return errors.New("provider is required")
	}
	return nil
}
