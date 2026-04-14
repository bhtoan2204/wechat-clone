// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type PinChatMessageRequest struct {
	RoomID    string `json:"room_id" form:"room_id" binding:"required"`
	MessageID string `json:"message_id" form:"message_id" binding:"required"`
}

func (r *PinChatMessageRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.MessageID = strings.TrimSpace(r.MessageID)
}

func (r *PinChatMessageRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	if r.MessageID == "" {
		return stackErr.Error(errors.New("message_id is required"))
	}
	return nil
}
