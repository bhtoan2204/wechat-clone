package support

import (
	"context"
	"errors"

	"go-socket/core/shared/pkg/actorctx"
)

func ActorFromCtx(ctx context.Context) (*actorctx.Actor, error) {
	actor, ok := actorctx.FromContext(ctx)
	if !ok || actor == nil {
		return nil, errors.New("unauthorized")
	}
	return actor, nil
}

func AccountIDFromCtx(ctx context.Context) (string, error) {
	actor, err := ActorFromCtx(ctx)
	if err != nil {
		return "", err
	}
	return actor.AccountID, nil
}
