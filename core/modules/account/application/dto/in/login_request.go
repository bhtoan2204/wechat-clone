// CODE_GENERATOR: request

package in

import (
	"errors"
	"strings"
)

type LoginRequest struct {
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required"`
}

func (r *LoginRequest) Normalize() {
	r.Email = strings.TrimSpace(r.Email)
	r.Password = strings.TrimSpace(r.Password)
}

func (r *LoginRequest) Validate() error {
	r.Normalize()
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}
