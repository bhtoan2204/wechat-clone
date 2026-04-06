// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type ForwardChatMessageRequest struct {
	MessageID    string `json:"message_id" form:"message_id" binding:"required"`
	TargetRoomID string `json:"target_room_id" form:"target_room_id" binding:"required"`
}

func (r *ForwardChatMessageRequest) Normalize() {
	r.MessageID = strings.TrimSpace(r.MessageID)
	r.TargetRoomID = strings.TrimSpace(r.TargetRoomID)
}

func (r *ForwardChatMessageRequest) Validate() error {
	r.Normalize()
	if r.MessageID == "" {
		return errors.New("message_id is required")
	}
	if r.TargetRoomID == "" {
		return errors.New("target_room_id is required")
	}
	return nil
}
