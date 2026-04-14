// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type DeleteChatMessageRequest struct {
	MessageID string `json:"message_id" form:"message_id" binding:"required"`
	Scope     string `json:"scope" form:"scope"`
}

func (r *DeleteChatMessageRequest) Normalize() {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.Scope = strings.TrimSpace(r.Scope)
}

func (r *DeleteChatMessageRequest) Validate() error {
	r.Normalize()
	if r.MessageID == "" {
		return stackErr.Error(errors.New("message_id is required"))
	}
	return nil
}
