// CODE_GENERATOR - do not edit: request

package in

import (
	"strings"
)

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
	UserAgent    string `json:"user_agent" form:"user_agent"`
	IpAddress    string `json:"ip_address" form:"ip_address"`
}

func (r *RefreshRequest) Normalize() {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	r.UserAgent = strings.TrimSpace(r.UserAgent)
	r.IpAddress = strings.TrimSpace(r.IpAddress)
}

func (r *RefreshRequest) Validate() error {
	r.Normalize()
	return nil
}
