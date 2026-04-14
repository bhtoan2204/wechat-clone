package repos

import (
	"context"
	"go-socket/core/modules/payment/domain/entity"
)

//go:generate mockgen -package=repos -destination=payment_account_projection_repo_mock.go -source=payment_account_projection_repo.go
type PaymentAccountProjectionRepository interface {
	GetAccountProjectionByAccountID(ctx context.Context, accountID string) (*entity.PaymentAccount, error)
	CreateAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error
	UpdateAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error
	DeleteAccountProjection(ctx context.Context, accountID string) error
	UpsertAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error
}
