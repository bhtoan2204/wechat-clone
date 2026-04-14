// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type SearchChatMentionsRequest struct {
	RoomID string `json:"room_id" form:"room_id" binding:"required"`
	Q      string `json:"q" form:"q"`
	Limit  int    `json:"limit" form:"limit"`
}

func (r *SearchChatMentionsRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.Q = strings.TrimSpace(r.Q)
}

func (r *SearchChatMentionsRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	return nil
}
