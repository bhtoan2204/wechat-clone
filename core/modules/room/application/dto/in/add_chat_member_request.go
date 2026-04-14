// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type AddChatMemberRequest struct {
	RoomID    string `json:"room_id" form:"room_id" binding:"required"`
	AccountID string `json:"account_id" form:"account_id" binding:"required"`
	Role      string `json:"role" form:"role"`
}

func (r *AddChatMemberRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.AccountID = strings.TrimSpace(r.AccountID)
	r.Role = strings.TrimSpace(r.Role)
}

func (r *AddChatMemberRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	if r.AccountID == "" {
		return stackErr.Error(errors.New("account_id is required"))
	}
	return nil
}
