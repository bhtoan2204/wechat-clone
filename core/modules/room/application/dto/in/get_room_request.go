// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
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
		return stackErr.Error(errors.New("id is required"))
	}
	return nil
}
