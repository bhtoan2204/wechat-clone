package repos

import (
	"context"
	"errors"

	"go-socket/core/modules/payment/domain/aggregate"
)

var ErrPaymentVersionConflict = errors.New("payment aggregate version conflict")

//go:generate mockgen -package=repos -destination=payment_balance_aggregate_repo_mock.go -source=payment_balance_aggregate_repo.go
type PaymentBalanceAggregateRepository interface {
	Load(ctx context.Context, accountID string) (*aggregate.PaymentBalanceAggregate, error)
	Save(ctx context.Context, aggregate *aggregate.PaymentBalanceAggregate) error
}
