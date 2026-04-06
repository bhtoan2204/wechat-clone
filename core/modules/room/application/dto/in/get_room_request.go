// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type GetRoomRequest struct {
	ID string `json:"id" form:"id" binding:"required"`
}

func (r *GetRoomRequest) Normalize() {
	r.ID = strings.TrimSpace(r.ID)
}

func (r *GetRoomRequest) Validate() error {
	r.Normalize()
	if r.ID == "" {
		return errors.New("id is required")
	}
	return nil
}
