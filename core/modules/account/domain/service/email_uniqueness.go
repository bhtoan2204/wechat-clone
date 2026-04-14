package service

import (
	"context"
	"errors"

	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/pkg/stackErr"
)

var ErrAccountEmailAlreadyExists = errors.New("account email already exists")

//go:generate mockgen -package=service -destination=email_uniqueness_mock.go -source=email_uniqueness.go
type EmailUniquenessChecker interface {
	IsEmailExists(ctx context.Context, email string) (bool, error)
}

func EnsureEmailAvailable(ctx context.Context, checker EmailUniquenessChecker, email valueobject.Email) error {
	exists, err := checker.IsEmailExists(ctx, email.Value())
	if err != nil {
		return stackErr.Error(err)
	}
	if exists {
		return stackErr.Error(ErrAccountEmailAlreadyExists)
	}
	return nil
}
