package projection

import (
	"context"
	"wechat-clone/core/modules/account/domain/entity"
)

type AccountReadRepository interface {
	GetAccountByID(ctx context.Context, id string) (*entity.Account, error)
	GetAccountByEmail(ctx context.Context, email string) (*entity.Account, error)
	SearchUsers(ctx context.Context, q string, limit, offset int) ([]*entity.Account, int64, error)
}
