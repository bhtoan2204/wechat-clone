// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"strings"
	"wechat-clone/core/shared/pkg/stackErr"
)

type GetChatMessageMediaRequest struct {
	RoomID    string `json:"room_id" form:"room_id" binding:"required"`
	ObjectKey string `json:"object_key" form:"object_key" binding:"required"`
}

func (r *GetChatMessageMediaRequest) Normalize() {
	r.RoomID = strings.TrimSpace(r.RoomID)
	r.ObjectKey = strings.TrimSpace(r.ObjectKey)
}

func (r *GetChatMessageMediaRequest) Validate() error {
	r.Normalize()
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	if r.ObjectKey == "" {
		return stackErr.Error(errors.New("object_key is required"))
	}
	return nil
}
