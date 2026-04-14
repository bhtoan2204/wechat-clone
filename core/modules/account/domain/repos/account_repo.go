package repos

import (
	"context"
	"go-socket/core/modules/account/domain/entity"
)

//go:generate mockgen -package=repos -destination=account_repo_mock.go -source=account_repo.go
type AccountRepository interface {
	GetAccountByID(ctx context.Context, id string) (*entity.Account, error)
	GetAccountByEmail(ctx context.Context, email string) (*entity.Account, error)
	IsEmailExists(ctx context.Context, email string) (bool, error)
	CreateAccount(ctx context.Context, account *entity.Account) error
	UpdateAccount(ctx context.Context, account *entity.Account) error
	DeleteAccount(ctx context.Context, id string) error
	ListAccountsByRoomID(ctx context.Context, roomID string) ([]*entity.Account, error)

	SearchUsers(ctx context.Context, q string, limit, offset int) ([]*entity.Account, int64, error)
}
