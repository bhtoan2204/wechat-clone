package in

import (
	"errors"
	"strings"
)

type AddChatMemberRequest struct {
	RoomID    string `json:"room_id" uri:"room_id"`
	AccountID string `json:"account_id"`
	Role      string `json:"role"`
}

func (r *AddChatMemberRequest) Validate() error {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.AccountID = strings.TrimSpace(r.AccountID)
	r.Role = strings.TrimSpace(r.Role)
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	if r.AccountID == "" {
		return errors.New("account_id is required")
	}
	return nil
}
