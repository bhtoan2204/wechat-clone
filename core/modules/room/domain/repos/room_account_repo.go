package repos

import (
	"context"
	"wechat-clone/core/modules/room/domain/entity"
)

//go:generate mockgen -package=repos -destination=account_repo_mock.go -source=account_repo.go
type RoomAccountRepository interface {
	ProjectAccount(context.Context, *entity.AccountEntity) error
	ListByAccountIDs(ctx context.Context, accountIDs []string) ([]*entity.AccountEntity, error)
}
