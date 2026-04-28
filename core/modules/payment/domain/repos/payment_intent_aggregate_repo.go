package repos

import (
	"context"
	aggregate "wechat-clone/core/modules/payment/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=payment_intent_aggregate_repo_mock.go -source=payment_intent_aggregate_repo.go
type PaymentIntentAggregateRepo interface {
	Save(ctx context.Context, aggregate *aggregate.PaymentIntentAggregate) error
	GetByTransactionID(ctx context.Context, transactionID string) (*aggregate.PaymentIntentAggregate, error)
	GetByExternalRef(ctx context.Context, provider, externalRef string) (*aggregate.PaymentIntentAggregate, error)
	ListPendingWithdrawals(ctx context.Context, limit int) ([]*aggregate.PaymentIntentAggregate, error)
}
