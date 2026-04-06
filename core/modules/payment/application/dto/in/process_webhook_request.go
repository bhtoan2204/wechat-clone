// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type ProcessWebhookRequest struct {
	Provider  string `json:"provider" form:"provider" binding:"required"`
	Signature string `json:"signature" form:"signature"`
	Payload   string `json:"payload" form:"payload"`
}

func (r *ProcessWebhookRequest) Normalize() {
	r.Provider = strings.TrimSpace(r.Provider)
	r.Signature = strings.TrimSpace(r.Signature)
	r.Payload = strings.TrimSpace(r.Payload)
}

func (r *ProcessWebhookRequest) Validate() error {
	r.Normalize()
	if r.Provider == "" {
		return errors.New("provider is required")
	}
	return nil
}
