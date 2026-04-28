package repository

import (
	"context"

	"wechat-clone/core/modules/payment/domain/aggregate"
	"wechat-clone/core/modules/payment/domain/repos"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type paymentIntentAggregateRepoImpl struct {
	providerPaymentRepo repos.ProviderPaymentRepository
}

func NewPaymentIntentAggregateRepo(db *gorm.DB) repos.PaymentIntentAggregateRepo {
	return newPaymentIntentAggregateRepo(newProviderPaymentRepoImpl(db))
}

func newPaymentIntentAggregateRepo(providerPaymentRepo repos.ProviderPaymentRepository) repos.PaymentIntentAggregateRepo {
	return &paymentIntentAggregateRepoImpl{
		providerPaymentRepo: providerPaymentRepo,
	}
}

func (r *paymentIntentAggregateRepoImpl) Save(ctx context.Context, aggregate *aggregate.PaymentIntentAggregate) error {
	if err := r.providerPaymentRepo.Save(ctx, aggregate); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *paymentIntentAggregateRepoImpl) GetByTransactionID(ctx context.Context, transactionID string) (*aggregate.PaymentIntentAggregate, error) {
	paymentAggregate, err := r.providerPaymentRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return paymentAggregate, nil
}

func (r *paymentIntentAggregateRepoImpl) GetByExternalRef(ctx context.Context, provider, externalRef string) (*aggregate.PaymentIntentAggregate, error) {
	paymentAggregate, err := r.providerPaymentRepo.GetByExternalRef(ctx, provider, externalRef)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return paymentAggregate, nil
}

func (r *paymentIntentAggregateRepoImpl) ListPendingWithdrawals(ctx context.Context, limit int) ([]*aggregate.PaymentIntentAggregate, error) {
	paymentAggregates, err := r.providerPaymentRepo.ListPendingWithdrawals(ctx, limit)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return paymentAggregates, nil
}
