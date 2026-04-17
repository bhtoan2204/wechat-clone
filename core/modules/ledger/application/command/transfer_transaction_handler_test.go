package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/service"
	"go-socket/core/modules/ledger/domain/entity"
	"go-socket/core/shared/infra/lock"
	"go-socket/core/shared/pkg/actorctx"

	"go.uber.org/mock/gomock"
)

func TestTransferTransactionHandle(t *testing.T) {
	t.Run("creates transfer transaction with sorted locks", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		ledgerService := service.NewMockLedgerService(ctrl)
		locker := lock.NewMockLock(ctrl)

		handler := &transferTransactionHandler{
			ledgerService: ledgerService,
			locker:        locker,
		}

		expectedTx := &entity.LedgerTransaction{
			TransactionID: "ledger-tx-1",
			Currency:      "VND",
			CreatedAt:     time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			Entries: []*entity.LedgerEntry{
				{
					ID:            10,
					TransactionID: "ledger-tx-1",
					AccountID:     "acc-z",
					Currency:      "VND",
					Amount:        -100,
					CreatedAt:     time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
				},
				{
					ID:            11,
					TransactionID: "ledger-tx-1",
					AccountID:     "acc-a",
					Currency:      "VND",
					Amount:        100,
					CreatedAt:     time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
				},
			},
		}

		var capturedCommand service.TransferToAccountCommand
		acquiredKeys := make([]string, 0, 2)
		releasedKeys := make([]string, 0, 2)

		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:acc-a", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			DoAndReturn(func(_ context.Context, key, _ string, _ time.Duration, _ time.Duration, _ time.Duration) (bool, error) {
				acquiredKeys = append(acquiredKeys, key)
				return true, nil
			})
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:acc-z", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			DoAndReturn(func(_ context.Context, key, _ string, _ time.Duration, _ time.Duration, _ time.Duration) (bool, error) {
				acquiredKeys = append(acquiredKeys, key)
				return true, nil
			})
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:acc-z", gomock.Any()).
			DoAndReturn(func(_ context.Context, key, _ string) (bool, error) {
				releasedKeys = append(releasedKeys, key)
				return true, nil
			})
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:acc-a", gomock.Any()).
			DoAndReturn(func(_ context.Context, key, _ string) (bool, error) {
				releasedKeys = append(releasedKeys, key)
				return true, nil
			})

		ledgerService.EXPECT().
			TransferToAccount(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, cmd service.TransferToAccountCommand) (*entity.LedgerTransaction, error) {
				capturedCommand = cmd
				return expectedTx, nil
			})

		ctx := actorctx.WithActor(context.Background(), actorctx.Actor{AccountID: "acc-z"})
		response, err := handler.Handle(ctx, &in.TransferTransactionRequest{
			ToAccountID: "acc-a",
			Currency:    "VND",
			Amount:      100,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if response.TransactionID != "ledger-tx-1" {
			t.Fatalf("expected transaction id ledger-tx-1, got %s", response.TransactionID)
		}
		if len(response.Entries) != 2 {
			t.Fatalf("expected 2 response entries, got %d", len(response.Entries))
		}

		if capturedCommand.FromAccountID != "acc-z" {
			t.Fatalf("expected from account acc-z, got %s", capturedCommand.FromAccountID)
		}
		if capturedCommand.ToAccountID != "acc-a" {
			t.Fatalf("expected to account acc-a, got %s", capturedCommand.ToAccountID)
		}
		if capturedCommand.Currency != "VND" {
			t.Fatalf("expected currency VND, got %s", capturedCommand.Currency)
		}
		if capturedCommand.Amount != 100 {
			t.Fatalf("expected amount 100, got %d", capturedCommand.Amount)
		}
		if capturedCommand.TransactionID == "" {
			t.Fatalf("expected generated transaction id")
		}
		if !stringSlicesEqual(acquiredKeys, []string{"ledger-account:acc-a", "ledger-account:acc-z"}) {
			t.Fatalf("unexpected acquired lock order: %v", acquiredKeys)
		}
		if !stringSlicesEqual(releasedKeys, []string{"ledger-account:acc-z", "ledger-account:acc-a"}) {
			t.Fatalf("unexpected released lock order: %v", releasedKeys)
		}
	})

	t.Run("rejects transfer when actor is missing", func(t *testing.T) {
		handler := &transferTransactionHandler{}

		_, err := handler.Handle(context.Background(), &in.TransferTransactionRequest{
			ToAccountID: "acc-a",
			Currency:    "VND",
			Amount:      100,
		})
		if !errors.Is(err, service.ErrUnauthorized) {
			t.Fatalf("expected unauthorized error, got %v", err)
		}
	})

	t.Run("returns insufficient funds from service", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		ledgerService := service.NewMockLedgerService(ctrl)
		handler := &transferTransactionHandler{
			ledgerService: ledgerService,
		}

		ledgerService.EXPECT().
			TransferToAccount(gomock.Any(), gomock.Any()).
			Return(nil, service.ErrInsufficientFunds)

		ctx := actorctx.WithActor(context.Background(), actorctx.Actor{AccountID: "acc-from"})
		_, err := handler.Handle(ctx, &in.TransferTransactionRequest{
			ToAccountID: "acc-to",
			Currency:    "USD",
			Amount:      100,
		})
		if !errors.Is(err, service.ErrInsufficientFunds) {
			t.Fatalf("expected insufficient funds error, got %v", err)
		}
	})
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}

	return true
}
