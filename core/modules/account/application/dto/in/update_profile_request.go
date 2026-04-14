// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type UpdateProfileRequest struct {
	DisplayName     string  `json:"display_name" form:"display_name" binding:"required"`
	Username        *string `json:"username" form:"username"`
	AvatarObjectKey *string `json:"avatar_object_key" form:"avatar_object_key"`
}

func (r *UpdateProfileRequest) Normalize() {
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	if r.Username != nil {
		*r.Username = strings.TrimSpace(*r.Username)
	}
	if r.AvatarObjectKey != nil {
		*r.AvatarObjectKey = strings.TrimSpace(*r.AvatarObjectKey)
	}
}

func (r *UpdateProfileRequest) Validate() error {
	r.Normalize()
	if r.DisplayName == "" {
		return stackErr.Error(errors.New("display_name is required"))
	}
	return nil
}
