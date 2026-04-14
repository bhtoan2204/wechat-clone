// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" form:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" form:"new_password" binding:"required"`
}

func (r *ChangePasswordRequest) Normalize() {
	r.CurrentPassword = strings.TrimSpace(r.CurrentPassword)
	r.NewPassword = strings.TrimSpace(r.NewPassword)
}

func (r *ChangePasswordRequest) Validate() error {
	r.Normalize()
	if r.CurrentPassword == "" {
		return stackErr.Error(errors.New("current_password is required"))
	}
	if r.NewPassword == "" {
		return stackErr.Error(errors.New("new_password is required"))
	}
	return nil
}
