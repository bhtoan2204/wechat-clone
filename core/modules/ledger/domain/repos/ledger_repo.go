package repos

import (
	"context"

	"go-socket/core/modules/ledger/domain/entity"
)

//go:generate mockgen -package=repos -destination=ledger_repo_mock.go -source=ledger_repo.go
type LedgerRepository interface {
	CreateTransaction(ctx context.Context, transaction *entity.LedgerTransaction) error
	InsertEntries(ctx context.Context, entries []*entity.LedgerEntry) error
	GetBalance(ctx context.Context, accountID, currency string) (int64, error)
	GetTransaction(ctx context.Context, transactionID string) (*entity.LedgerTransaction, error)
}
