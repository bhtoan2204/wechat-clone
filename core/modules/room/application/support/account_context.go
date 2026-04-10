package support

import (
	"context"

	"go-socket/core/shared/pkg/actorctx"
)

func AccountIDFromCtx(ctx context.Context) (string, error) {
	return actorctx.AccountIDFromContext(ctx)
}
