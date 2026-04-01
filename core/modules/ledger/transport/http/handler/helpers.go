package handler

import (
	"context"
	"errors"

	"go-socket/core/shared/infra/xpaseto"
)

func accountIDFromContext(ctx context.Context) (string, error) {
	account, ok := ctx.Value("account").(*xpaseto.PasetoPayload)
	if !ok || account == nil || account.AccountID == "" {
		return "", errors.New("account not found")
	}
	return account.AccountID, nil
}
