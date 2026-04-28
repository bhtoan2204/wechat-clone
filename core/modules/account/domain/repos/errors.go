package repos

import "errors"

var (
	ErrAccountEmailAlreadyExists    = errors.New("account email already exists")
	ErrAccountUsernameAlreadyExists = errors.New("account username already exists")
)
