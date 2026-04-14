// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type MarkChatMessageStatusRequest struct {
	MessageID string `json:"message_id" form:"message_id" binding:"required"`
	Status    string `json:"status" form:"status" binding:"required"`
}

func (r *MarkChatMessageStatusRequest) Normalize() {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.Status = strings.TrimSpace(r.Status)
}

func (r *MarkChatMessageStatusRequest) Validate() error {
	r.Normalize()
	if r.MessageID == "" {
		return stackErr.Error(errors.New("message_id is required"))
	}
	if r.Status == "" {
		return stackErr.Error(errors.New("status is required"))
	}
	return nil
}
