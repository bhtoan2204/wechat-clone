package support

import (
	"context"
	"errors"

	"go-socket/core/shared/pkg/actorctx"
	"go-socket/core/shared/pkg/stackErr"
)

func ActorFromCtx(ctx context.Context) (*actorctx.Actor, error) {
	actor, ok := actorctx.FromContext(ctx)
	if !ok || actor == nil {
		return nil, stackErr.Error(errors.New("unauthorized"))
	}
	return actor, nil
}

func AccountIDFromCtx(ctx context.Context) (string, error) {
	actor, err := ActorFromCtx(ctx)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return actor.AccountID, nil
}
