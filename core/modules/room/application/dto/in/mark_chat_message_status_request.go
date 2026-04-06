// CODE_GENERATOR: request

package in

import (
	"errors"
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
		return errors.New("message_id is required")
	}
	if r.Status == "" {
		return errors.New("status is required")
	}
	return nil
}
