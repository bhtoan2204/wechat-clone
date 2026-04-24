// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"strings"
	"wechat-clone/core/shared/pkg/stackErr"
)

type ProcessWebhookRequest struct {
	Provider  string `json:"provider" form:"provider" binding:"required"`
	Signature string `json:"signature" form:"signature"`
	Payload   string `json:"payload" form:"payload"`
}

func (r *ProcessWebhookRequest) Normalize() {
	r.Provider = strings.TrimSpace(r.Provider)
	r.Signature = strings.TrimSpace(r.Signature)
}

func (r *ProcessWebhookRequest) Validate() error {
	r.Normalize()
	if r.Provider == "" {
		return stackErr.Error(errors.New("provider is required"))
	}
	return nil
}
