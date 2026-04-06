// CODE_GENERATOR: request

package in

import (
	"errors"
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
		return errors.New("mode is required")
	}
	return nil
}
