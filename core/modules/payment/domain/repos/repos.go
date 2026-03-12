package repos

import (
	"context"
)

type Repos interface {
	PaymentOutboxEventsRepository() PaymentOutboxEventsRepository
	PaymentAccountProjectionRepository() PaymentAccountProjectionRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
