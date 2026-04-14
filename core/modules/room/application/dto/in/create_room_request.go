// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type CreateRoomRequest struct {
	Name        string `json:"name" form:"name" binding:"required"`
	Description string `json:"description" form:"description"`
	RoomType    string `json:"room_type" form:"room_type"`
}

func (r *CreateRoomRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.RoomType = strings.TrimSpace(r.RoomType)
}

func (r *CreateRoomRequest) Validate() error {
	r.Normalize()
	if r.Name == "" {
		return stackErr.Error(errors.New("name is required"))
	}
	return nil
}
