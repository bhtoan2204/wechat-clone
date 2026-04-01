package repos

import "context"

type Repos interface {
	LedgerRepository() LedgerRepository
	PaymentRepository() PaymentRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
