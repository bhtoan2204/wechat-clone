package repos

import (
	"context"
	"go-socket/core/modules/payment/domain/entity"
	"go-socket/core/shared/utils"
)

//go:generate mockgen -package=repos -destination=payment_history_repo_mock.go -source=payment_history_repo.go
type PaymentHistoryRepository interface {
	ListPaymentHistory(ctx context.Context, options utils.QueryOptions) ([]*entity.PaymentHistory, error)
}
