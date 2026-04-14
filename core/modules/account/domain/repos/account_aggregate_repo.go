package repos

import (
	"context"
	"go-socket/core/modules/account/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=account_aggregate_repo_mock.go -source=account_aggregate_repo.go
type AccountAggregateRepository interface {
	Load(ctx context.Context, accountID string) (*aggregate.AccountAggregate, error)
	LoadByEmail(ctx context.Context, email string) (*aggregate.AccountAggregate, error)
	Save(ctx context.Context, agg *aggregate.AccountAggregate) error
}
