// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type ListChatMessagesRequest struct {
	RoomID    string `json:"room_id" form:"room_id" binding:"required"`
	Limit     int    `json:"limit" form:"limit"`
	BeforeID  string `json:"before_id" form:"before_id"`
	BeforeAt  string `json:"before_at" form:"before_at"`
	Ascending bool   `json:"ascending" form:"ascending"`
}

func (r *ListChatMessagesRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.BeforeID = strings.TrimSpace(r.BeforeID)
	r.BeforeAt = strings.TrimSpace(r.BeforeAt)
}

func (r *ListChatMessagesRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	return nil
}
