package repos

import (
	"context"
)

//go:generate mockgen -package=repos -destination=repos_mock.go -source=repos.go
type Repos interface {
	PaymentBalanceAggregateRepository() PaymentBalanceAggregateRepository
	PaymentProjectionRepository() PaymentProjectionRepository
	PaymentOutboxEventsRepository() PaymentOutboxEventsRepository
	PaymentAccountProjectionRepository() PaymentAccountProjectionRepository
	PaymentHistoryRepository() PaymentHistoryRepository
	ProviderPaymentRepository() ProviderPaymentRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
