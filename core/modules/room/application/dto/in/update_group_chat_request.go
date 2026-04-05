package in

import (
	"errors"
	"strings"
)

type UpdateGroupChatRequest struct {
	RoomID      string `json:"room_id" uri:"room_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *UpdateGroupChatRequest) Validate() error {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	return nil
}
