package in

import (
	"errors"
	"strings"
)

type ForwardChatMessageRequest struct {
	MessageID    string `json:"message_id" uri:"message_id"`
	TargetRoomID string `json:"target_room_id"`
}

func (r *ForwardChatMessageRequest) Validate() error {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.TargetRoomID = strings.TrimSpace(r.TargetRoomID)
	if r.MessageID == "" {
		return errors.New("message_id is required")
	}
	if r.TargetRoomID == "" {
		return errors.New("target_room_id is required")
	}
	return nil
}
