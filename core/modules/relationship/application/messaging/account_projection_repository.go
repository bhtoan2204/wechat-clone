package messaging

import (
	"context"

	"wechat-clone/core/modules/relationship/domain/entity"
)

//go:generate mockgen -package=messaging -destination=account_projection_repository_mock.go -source=account_projection_repository.go
type AccountProjectionRepository interface {
	ProjectAccount(ctx context.Context, account *entity.AccountProjection) error
	GetByID(ctx context.Context, accountID string) (*entity.AccountProjection, error)
}
