package in

import (
	"errors"
	"strings"
)

type DeleteChatMessageRequest struct {
	MessageID string `json:"message_id" uri:"message_id"`
	Scope     string `json:"scope" form:"scope"`
}

func (r *DeleteChatMessageRequest) Validate() error {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.Scope = strings.TrimSpace(r.Scope)
	if r.MessageID == "" {
		return errors.New("message_id is required")
	}
	return nil
}
