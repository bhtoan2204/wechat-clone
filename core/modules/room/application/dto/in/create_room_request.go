// CODE_GENERATOR: request

package in

import (
	"errors"
	"go-socket/core/modules/room/types"
	"strings"
)

type CreateRoomRequest struct {
	Name        string         `json:"name" form:"name"`
	Description string         `json:"description" form:"description"`
	RoomType    types.RoomType `json:"room_type" form:"room_type"`
}

func (r *CreateRoomRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.RoomType = r.RoomType.Normalize()

	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.RoomType == "" {
		r.RoomType = types.RoomTypePublic
	}
	if !r.RoomType.IsValid() {
		return errors.New("room_type must be one of: public, private, direct, group")
	}

	return nil
}
