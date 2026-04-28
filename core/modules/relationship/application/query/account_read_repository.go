package query

import (
	"context"

	"wechat-clone/core/modules/relationship/domain/entity"
)

//go:generate mockgen -package=query -destination=account_read_repository_mock.go -source=account_read_repository.go
type AccountReadRepository interface {
	GetByID(ctx context.Context, accountID string) (*entity.AccountProjection, error)
}
