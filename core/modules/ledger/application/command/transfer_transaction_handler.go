package command

import (
	"context"
	"fmt"
	"time"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/dto/out"
	ledgerservice "go-socket/core/modules/ledger/application/service"
	"go-socket/core/shared/infra/lock"
	"go-socket/core/shared/pkg/actorctx"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
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
		return nil, stackErr.Error(fmt.Errorf("%w: %w", ledgerservice.ErrUnauthorized, err))
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

	opts := lock.DefaultMultiLockOptions()
	opts.KeyPrefix = ledgerservice.LedgerAccountLockKeyPrefix

	response, err := lock.WithLocks(
		ctx,
		u.locker,
		[]string{fromAccountID, req.ToAccountID},
		opts,
		transferFn,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return response, nil
}
