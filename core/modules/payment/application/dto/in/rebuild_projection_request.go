// CODE_GENERATOR - do not edit: request

package in

import (
	"errors"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
)

type RebuildProjectionRequest struct {
	Mode      string `json:"mode" form:"mode" binding:"required"`
	AccountID string `json:"account_id" form:"account_id"`
}

func (r *RebuildProjectionRequest) Normalize() {
	r.Mode = strings.TrimSpace(r.Mode)
	r.AccountID = strings.TrimSpace(r.AccountID)
}

func (r *RebuildProjectionRequest) Validate() error {
	r.Normalize()
	if r.Mode == "" {
		return stackErr.Error(errors.New("mode is required"))
	}
	return nil
}
