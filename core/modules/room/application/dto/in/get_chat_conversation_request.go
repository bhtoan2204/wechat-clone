package in

import (
	"errors"
	"strings"
)

type GetChatConversationRequest struct {
	RoomID string `json:"room_id" uri:"room_id"`
}

func (r *GetChatConversationRequest) Validate() error {
	r.RoomID = strings.TrimSpace(r.RoomID)
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	return nil
}
