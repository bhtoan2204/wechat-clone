package in

import (
	"errors"
	"strings"
)

type MarkChatMessageStatusRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Status    string `json:"status"`
}

func (r *MarkChatMessageStatusRequest) Validate() error {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.Status = strings.TrimSpace(r.Status)
	if r.MessageID == "" {
		return errors.New("message_id is required")
	}
	if r.Status == "" {
		return errors.New("status is required")
	}
	return nil
}
