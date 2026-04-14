// CODE_GENERATOR: request

package in

import (
	"strings"
)

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
}

func (r *RefreshRequest) Normalize() {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
}

func (r *RefreshRequest) Validate() error {
	r.Normalize()
	return nil
}
