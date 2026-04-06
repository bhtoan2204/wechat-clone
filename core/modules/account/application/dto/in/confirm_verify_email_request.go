// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type ConfirmVerifyEmailRequest struct {
	Token string `json:"token" form:"token" binding:"required"`
}

func (r *ConfirmVerifyEmailRequest) Normalize() {
	r.Token = strings.TrimSpace(r.Token)
}

func (r *ConfirmVerifyEmailRequest) Validate() error {
	r.Normalize()
	if r.Token == "" {
		return errors.New("token is required")
	}
	return nil
}
