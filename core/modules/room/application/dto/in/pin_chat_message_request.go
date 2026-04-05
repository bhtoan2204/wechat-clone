package in

import (
	"errors"
	"strings"
)

type PinChatMessageRequest struct {
	RoomID    string `json:"room_id" uri:"room_id"`
	MessageID string `json:"message_id"`
}

func (r *PinChatMessageRequest) Validate() error {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.MessageID = strings.TrimSpace(r.MessageID)
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	if r.MessageID == "" {
		return errors.New("message_id is required")
	}
	return nil
}
