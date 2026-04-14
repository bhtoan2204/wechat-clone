// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type UpdateRoomRequest struct {
	ID   string `json:"id" form:"id" binding:"required"`
	Name string `json:"name" form:"name" binding:"required"`
}

func (r *UpdateRoomRequest) Normalize() {
	r.ID = strings.TrimSpace(r.ID)
	r.Name = strings.TrimSpace(r.Name)
}

func (r *UpdateRoomRequest) Validate() error {
	r.Normalize()
	if r.ID == "" {
		return stackErr.Error(errors.New("id is required"))
	}
	if r.Name == "" {
		return stackErr.Error(errors.New("name is required"))
	}
	return nil
}
