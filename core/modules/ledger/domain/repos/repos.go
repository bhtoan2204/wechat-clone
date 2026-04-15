package repos

import "context"

//go:generate mockgen -package=repos -destination=repos_mock.go -source=repos.go
type Repos interface {
	LedgerRepository() LedgerRepository
	LedgerAccountAggregateRepository() LedgerAccountAggregateRepository
	LedgerOutboxEventsRepository() LedgerOutboxEventsRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
