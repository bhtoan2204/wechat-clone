package service

import (
	"context"

	"go-socket/core/modules/account/domain/rules"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/pkg/stackErr"
)

type PasswordReuseChecker interface {
	Verify(ctx context.Context, val string, hash string) (bool, error)
}

func EnsurePasswordIsNew(
	ctx context.Context,
	checker PasswordReuseChecker,
	newPassword valueobject.PlainPassword,
	currentHash valueobject.HashedPassword,
) error {
	isSamePassword, err := checker.Verify(ctx, newPassword.Value(), currentHash.Value())
	if err != nil {
		return stackErr.Error(err)
	}
	if isSamePassword {
		return stackErr.Error(rules.ErrAccountPasswordSameAsOldOne)
	}
	return nil
}
