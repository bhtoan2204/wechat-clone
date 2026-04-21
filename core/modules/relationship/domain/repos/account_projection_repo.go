package repos

import (
	"context"
	"wechat-clone/core/modules/relationship/domain/entity"
)

type RelationshipAccountRepository interface {
	ProjectAccount(ctx context.Context, account *entity.AccountProjection) error
	GetByID(ctx context.Context, accountID string) (*entity.AccountProjection, error)
	Exists(ctx context.Context, accountID string) (bool, error)
}
