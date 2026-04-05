package support

import (
	"context"
	"errors"

	"go-socket/core/shared/infra/xpaseto"
)

func AccountIDFromCtx(ctx context.Context) (string, error) {
	payload, ok := ctx.Value("account").(*xpaseto.PasetoPayload)
	if !ok || payload == nil || payload.AccountID == "" {
		return "", errors.New("unauthorized")
	}
	return payload.AccountID, nil
}
