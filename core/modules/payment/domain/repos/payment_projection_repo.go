package repos

import (
	"context"
	"time"

	"go-socket/core/modules/payment/domain/types"
)

type ProjectionRebuildMode string

const (
	ProjectionRebuildModeFull     ProjectionRebuildMode = "full"
	ProjectionRebuildModeSnapshot ProjectionRebuildMode = "snapshot"
)

type ProjectionRebuildResult struct {
	Accounts            int
	EventsReplayed      int
	TransactionsRebuilt int
	BalancesRebuilt     int
}

//go:generate mockgen -package=repos -destination=payment_projection_repo_mock.go -source=payment_projection_repo.go
type PaymentProjectionRepository interface {
	ProjectTransaction(ctx context.Context, eventID, transactionID, accountID string, amount, balanceDelta int64, transactionType types.TransactionType, createdAt time.Time) error
	RebuildProjection(ctx context.Context, accountID string, mode ProjectionRebuildMode) (*ProjectionRebuildResult, error)
}
