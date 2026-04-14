package repos

import "context"

//go:generate mockgen -package=repos -destination=repos_mock.go -source=repos.go
type Repos interface {
	AccountRepository() AccountRepository
	AccountAggregateRepository() AccountAggregateRepository
	DeviceRepository() DeviceRepository
	SessionRepository() SessionRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
