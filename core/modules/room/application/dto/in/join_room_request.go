package in

import (
	"errors"

	"go-socket/core/shared/pkg/stackErr"
)

type JoinRoomRequest struct {
	RoomID string `json:"room_id" form:"room_id"`
}

func (r *JoinRoomRequest) Validate() error {
	if r.RoomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}
	return nil
}
