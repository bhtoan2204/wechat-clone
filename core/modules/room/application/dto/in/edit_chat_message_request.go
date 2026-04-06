// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type EditChatMessageRequest struct {
	MessageID string `json:"message_id" form:"message_id" binding:"required"`
	Message   string `json:"message" form:"message" binding:"required"`
}

func (r *EditChatMessageRequest) Normalize() {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.Message = strings.TrimSpace(r.Message)
}

func (r *EditChatMessageRequest) Validate() error {
	r.Normalize()
	if r.MessageID == "" {
		return errors.New("message_id is required")
	}
	if r.Message == "" {
		return errors.New("message is required")
	}
	return nil
}
