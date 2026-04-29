package command

import (
	"errors"
)

var (
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrAccountExists             = errors.New("account already exists")
	ErrUsernameExists            = errors.New("username already exists")
	ErrAccountNotFound           = errors.New("account not found")
	ErrCheckAccountFailed        = errors.New("check account failed")
	ErrInvalidCurrentPassword    = errors.New("current password is invalid")
	ErrInvalidVerificationToken  = errors.New("verification token is invalid or expired")
	ErrRegistrationAccountExists = errors.New("registration account already exists")
	ErrRefreshTokenInvalid       = errors.New("refresh token is invalid")
	ErrRefreshSessionExpired     = errors.New("refresh session expired")
	ErrRefreshSessionRevoked     = errors.New("refresh session revoked")
)
