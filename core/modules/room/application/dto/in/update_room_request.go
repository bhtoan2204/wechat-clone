// CODE_GENERATOR: request

package in

import (
	"errors"
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
		return errors.New("id is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}
