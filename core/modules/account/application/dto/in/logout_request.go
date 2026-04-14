// CODE_GENERATOR - do not edit: request

package in

import (
	"strings"
)

type LogoutRequest struct {
	Token string `json:"token" form:"token"`
}

func (r *LogoutRequest) Normalize() {
	r.Token = strings.TrimSpace(r.Token)
}

func (r *LogoutRequest) Validate() error {
	r.Normalize()
	return nil
}
