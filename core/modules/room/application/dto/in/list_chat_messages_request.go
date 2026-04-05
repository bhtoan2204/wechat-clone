package in

import (
	"errors"
	"strings"
)

type ListChatMessagesRequest struct {
	RoomID    string `json:"room_id" uri:"room_id"`
	Limit     int    `form:"limit"`
	BeforeID  string `form:"before_id"`
	BeforeAt  string `form:"before_at"`
	Ascending bool   `form:"ascending"`
}

func (r *ListChatMessagesRequest) Validate() error {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.BeforeID = strings.TrimSpace(r.BeforeID)
	r.BeforeAt = strings.TrimSpace(r.BeforeAt)
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	if r.Limit < 0 {
		r.Limit = 0
	}
	return nil
}
