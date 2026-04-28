package repos

import (
	"context"

	paymentaggregate "wechat-clone/core/modules/payment/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=provider_payment_repo_mock.go -source=provider_payment_repo.go
type ProviderPaymentRepository interface {
	Save(ctx context.Context, aggregate *paymentaggregate.PaymentIntentAggregate) error
	GetByTransactionID(ctx context.Context, transactionID string) (*paymentaggregate.PaymentIntentAggregate, error)
	GetByExternalRef(ctx context.Context, provider, externalRef string) (*paymentaggregate.PaymentIntentAggregate, error)
	ListPendingWithdrawals(ctx context.Context, limit int) ([]*paymentaggregate.PaymentIntentAggregate, error)
}
