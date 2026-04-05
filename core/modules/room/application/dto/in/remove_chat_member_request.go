package in

import (
	"errors"
	"strings"
)

type RemoveChatMemberRequest struct {
	RoomID    string `json:"room_id" uri:"room_id"`
	AccountID string `json:"account_id" uri:"account_id"`
}

func (r *RemoveChatMemberRequest) Validate() error {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.AccountID = strings.TrimSpace(r.AccountID)
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	if r.AccountID == "" {
		return errors.New("account_id is required")
	}
	return nil
}
