// CODE_GENERATOR: request
package in

import (
	"errors"
	"strings"

	"go-socket/core/shared/pkg/stackErr"
)

type SearchChatMentionsRequest struct {
	RoomID string `json:"room_id" form:"room_id" binding:"required"`
	Query  string `json:"q" form:"q"`
	Limit  int    `json:"limit" form:"limit"`
}

func (r *SearchChatMentionsRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.Query = strings.TrimSpace(r.Query)
}

func (r *SearchChatMentionsRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	return nil
}
