package actorctx

import (
	"context"
	"errors"
	"strings"
)

type Actor struct {
	AccountID string
	Email     string
	Role      string
}

type contextKey struct{}

func WithActor(ctx context.Context, actor Actor) context.Context {
	actor.AccountID = strings.TrimSpace(actor.AccountID)
	actor.Email = strings.TrimSpace(actor.Email)
	actor.Role = strings.TrimSpace(actor.Role)
	return context.WithValue(ctx, contextKey{}, actor)
}

func FromContext(ctx context.Context) (*Actor, bool) {
	actor, ok := ctx.Value(contextKey{}).(Actor)
	if !ok || actor.AccountID == "" {
		return nil, false
	}
	return &actor, true
}

func AccountIDFromContext(ctx context.Context) (string, error) {
	actor, ok := FromContext(ctx)
	if !ok {
		return "", errors.New("unauthorized")
	}
	return actor.AccountID, nil
}
