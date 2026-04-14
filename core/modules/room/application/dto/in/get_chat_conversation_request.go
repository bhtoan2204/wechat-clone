// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type GetChatConversationRequest struct {
	RoomID string `json:"room_id" form:"room_id" binding:"required"`
}

func (r *GetChatConversationRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
}

func (r *GetChatConversationRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	return nil
}
