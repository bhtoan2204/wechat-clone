package repos

import (
	"context"
	"go-socket/core/modules/payment/domain/entity"
)

type PaymentAccountProjectionRepository interface {
	GetAccountProjectionByAccountID(ctx context.Context, accountID string) (*entity.PaymentAccount, error)
	CreateAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error
	UpdateAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error
	DeleteAccountProjection(ctx context.Context, accountID string) error
}
