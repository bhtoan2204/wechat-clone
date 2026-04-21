package repos

import (
	"context"

	"wechat-clone/core/modules/ledger/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=ledger_account_aggregate_repo_mock.go -source=ledger_account_aggregate_repo.go
type LedgerAccountAggregateRepository interface {
	Load(ctx context.Context, accountID string) (*aggregate.LedgerAccountAggregate, error)
	Save(ctx context.Context, aggregate *aggregate.LedgerAccountAggregate) error
}
