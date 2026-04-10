package command

import (
	"context"

	"go-socket/core/shared/pkg/actorctx"
)

func accountIDFromContext(ctx context.Context) (string, error) {
	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		return "", ErrPaymentAccountNotFound
	}

	return accountID, nil
}
