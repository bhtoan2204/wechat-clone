package repos

import (
	"context"

	"go-socket/core/modules/ledger/domain/entity"
)

// LedgerRepository exposes read-side ledger views derived from canonical
// transaction postings. Write-side persistence must go through aggregate
// repositories to keep the posting model explicit.
//
//go:generate mockgen -package=repos -destination=ledger_repo_mock.go -source=ledger_repo.go
type LedgerRepository interface {
	GetBalance(ctx context.Context, accountID, currency string) (int64, error)
	GetTransaction(ctx context.Context, transactionID string) (*entity.LedgerTransaction, error)
}
