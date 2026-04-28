package messaging

import (
	"context"

	"wechat-clone/core/modules/room/domain/entity"
)

//go:generate mockgen -package=messaging -destination=account_projection_repository_mock.go -source=account_projection_repository.go
type AccountProjectionRepository interface {
	ProjectAccount(context.Context, *entity.AccountEntity) error
}
