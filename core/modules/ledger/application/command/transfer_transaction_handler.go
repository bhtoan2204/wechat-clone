package command

import (
	"context"
	"fmt"
	"sort"
	"time"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/infra/lock"
	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/dto/out"
	ledgerservice "go-socket/core/modules/ledger/application/service"
	"go-socket/core/shared/pkg/actorctx"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type transferTransactionHandler struct {
	ledgerService ledgerservice.LedgerService
	locker        lock.Lock
}

func NewTransferTransaction(
	appCtx *appCtx.AppContext,
	ledgerService ledgerservice.LedgerService,
) cqrs.Handler[*in.TransferTransactionRequest, *out.TransactionTransactionResponse] {
	return &transferTransactionHandler{
		ledgerService: ledgerService,
		locker:        appCtx.Locker(),
	}
}

func (u *transferTransactionHandler) Handle(ctx context.Context, req *in.TransferTransactionRequest) (*out.TransactionTransactionResponse, error) {
	fromAccountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%v: %v", ledgerservice.ErrUnauthorized, err))
	}

	transferFn := func() (*out.TransactionTransactionResponse, error) {
		transaction, err := u.ledgerService.TransferToAccount(ctx, ledgerservice.TransferToAccountCommand{
			TransactionID: uuid.NewString(),
			FromAccountID: fromAccountID,
			ToAccountID:   req.ToAccountID,
			Currency:      req.Currency,
			Amount:        req.Amount,
		})
		if err != nil {
			return nil, stackErr.Error(err)
		}

		responseEntries := make([]out.LedgerEntryResponse, 0, len(transaction.Entries))
		for _, entry := range transaction.Entries {
			responseEntries = append(responseEntries, out.LedgerEntryResponse{
				ID:            entry.ID,
				TransactionID: entry.TransactionID,
				AccountID:     entry.AccountID,
				Currency:      entry.Currency,
				Amount:        entry.Amount,
				CreatedAt:     entry.CreatedAt,
			})
		}

		return &out.TransactionTransactionResponse{
			TransactionID: transaction.TransactionID,
			Currency:      transaction.Currency,
			CreatedAt:     transaction.CreatedAt.UTC().Format(time.RFC3339Nano),
			Entries:       responseEntries,
		}, nil
	}

	response, err := u.withTransferLocks(ctx, fromAccountID, req.ToAccountID, transferFn)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return response, nil
}

func (u *transferTransactionHandler) withTransferLocks(
	ctx context.Context,
	leftAccountID string,
	rightAccountID string,
	fn func() (*out.TransactionTransactionResponse, error),
) (*out.TransactionTransactionResponse, error) {
	if u.locker == nil {
		return fn()
	}

	lockKeys := []string{
		fmt.Sprintf("ledger-transfer:%s", leftAccountID),
		fmt.Sprintf("ledger-transfer:%s", rightAccountID),
	}
	sort.Strings(lockKeys)

	lockValue := uuid.NewString()
	releaseKeys := make([]string, 0, len(lockKeys))
	for _, lockKey := range lockKeys {
		locked, err := u.locker.AcquireLock(ctx, lockKey, lockValue, 30*time.Second, 100*time.Millisecond, 3*time.Second)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		if !locked {
			return nil, stackErr.Error(fmt.Errorf("acquire transfer lock failed: %s", lockKey))
		}
		releaseKeys = append(releaseKeys, lockKey)
	}

	defer func() {
		for idx := len(releaseKeys) - 1; idx >= 0; idx-- {
			if _, err := u.locker.ReleaseLock(ctx, releaseKeys[idx], lockValue); err != nil {
				logging.FromContext(ctx).Warnw("release transfer lock failed", zap.String("lock_key", releaseKeys[idx]), zap.Error(err))
			}
		}
	}()

	return fn()
}
