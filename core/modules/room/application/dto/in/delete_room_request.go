// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type DeleteRoomRequest struct {
	ID string `json:"id" form:"id" binding:"required"`
}

func (r *DeleteRoomRequest) Normalize() {
	r.ID = strings.TrimSpace(r.ID)
}

func (r *DeleteRoomRequest) Validate() error {
	r.Normalize()
	if r.ID == "" {
		return errors.New("id is required")
	}
	return nil
}
