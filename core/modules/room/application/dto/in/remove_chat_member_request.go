// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type RemoveChatMemberRequest struct {
	RoomID    string `json:"room_id" form:"room_id" binding:"required"`
	AccountID string `json:"account_id" form:"account_id" binding:"required"`
}

func (r *RemoveChatMemberRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.AccountID = strings.TrimSpace(r.AccountID)
}

func (r *RemoveChatMemberRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return errors.New("room_id is required")
	}
	if r.AccountID == "" {
		return errors.New("account_id is required")
	}
	return nil
}
