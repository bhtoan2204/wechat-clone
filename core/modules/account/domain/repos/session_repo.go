package repos

import (
	"context"

	"go-socket/core/modules/account/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=session_repo_mock.go -source=session_repo.go
type SessionRepository interface {
	Load(ctx context.Context, sessionID string) (*aggregate.SessionAggregate, error)
	Save(ctx context.Context, session *aggregate.SessionAggregate) error
	ListByAccountID(ctx context.Context, accountID string) ([]*aggregate.SessionAggregate, error)
}
