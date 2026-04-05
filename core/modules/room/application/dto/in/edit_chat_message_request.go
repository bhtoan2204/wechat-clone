package in

import (
	"errors"
	"strings"
)

type EditChatMessageRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Message   string `json:"message"`
}

func (r *EditChatMessageRequest) Validate() error {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.Message = strings.TrimSpace(r.Message)
	if r.MessageID == "" {
		return errors.New("message_id is required")
	}
	if r.Message == "" {
		return errors.New("message is required")
	}
	return nil
}
