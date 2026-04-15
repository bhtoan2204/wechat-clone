package service

import (
	"context"
	"errors"
	"testing"
	"time"

	ledgerprojection "go-socket/core/modules/ledger/application/projection"
	ledgeraggregate "go-socket/core/modules/ledger/domain/aggregate"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	eventpkg "go-socket/core/shared/pkg/event"

	"go.uber.org/mock/gomock"
)

func TestLedgerServiceTransferToAccount(t *testing.T) {
	t.Run("posts transfer across account streams and outbox", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)
		outboxRepo := ledgerrepos.NewMockLedgerOutboxEventsRepository(ctrl)

		fromAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("acc-from")
		fromAgg.Balances["VND"] = 500

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		txRepos.EXPECT().LedgerOutboxEventsRepository().Return(outboxRepo)

		accountRepo.EXPECT().Load(gomock.Any(), "acc-from").Return(fromAgg, nil)
		accountRepo.EXPECT().Load(gomock.Any(), "acc-to").Return(nil, nil)
		accountRepo.EXPECT().
			Save(gomock.Any(), gomock.AssignableToTypeOf(&ledgeraggregate.LedgerAccountAggregate{})).
			DoAndReturn(func(_ context.Context, aggregate *ledgeraggregate.LedgerAccountAggregate) error {
				switch aggregate.AggregateID() {
				case "acc-from":
					if aggregate.Balance("VND") != 400 {
						t.Fatalf("expected acc-from balance 400, got %d", aggregate.Balance("VND"))
					}
				case "acc-to":
					if aggregate.Balance("VND") != 100 {
						t.Fatalf("expected acc-to balance 100, got %d", aggregate.Balance("VND"))
					}
				default:
					t.Fatalf("unexpected aggregate saved: %s", aggregate.AggregateID())
				}
				return nil
			}).Times(2)
		outboxRepo.EXPECT().
			Append(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, evt eventpkg.Event) error {
				if evt.EventName != ledgerprojection.EventLedgerTransactionProjected {
					t.Fatalf("unexpected outbox event name: %s", evt.EventName)
				}
				payload, ok := evt.EventData.(*ledgerprojection.LedgerTransactionProjected)
				if !ok {
					t.Fatalf("unexpected outbox payload type: %T", evt.EventData)
				}
				if payload.TransactionID != "ledger-tx-1" {
					t.Fatalf("unexpected transaction id: %s", payload.TransactionID)
				}
				if len(payload.Entries) != 2 {
					t.Fatalf("expected 2 projected entries, got %d", len(payload.Entries))
				}
				return nil
			})

		service := NewLedgerService(baseRepo)

		transaction, err := service.TransferToAccount(context.Background(), TransferToAccountCommand{
			TransactionID: "ledger-tx-1",
			FromAccountID: "acc-from",
			ToAccountID:   "acc-to",
			Currency:      "VND",
			Amount:        100,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if transaction.TransactionID != "ledger-tx-1" {
			t.Fatalf("unexpected transaction id: %s", transaction.TransactionID)
		}
		if transaction.Currency != "VND" {
			t.Fatalf("unexpected currency: %s", transaction.Currency)
		}
		if len(transaction.Entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(transaction.Entries))
		}
	})

	t.Run("rejects insufficient funds from account aggregate state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)

		fromAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("acc-from")
		fromAgg.Balances["USD"] = 50

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "acc-from").Return(fromAgg, nil)
		accountRepo.EXPECT().Load(gomock.Any(), "acc-to").Return(nil, nil)

		service := NewLedgerService(baseRepo)
		_, err := service.TransferToAccount(context.Background(), TransferToAccountCommand{
			TransactionID: "ledger-tx-2",
			FromAccountID: "acc-from",
			ToAccountID:   "acc-to",
			Currency:      "USD",
			Amount:        100,
		})
		if !errors.Is(err, ErrInsufficientFunds) {
			t.Fatalf("expected insufficient funds error, got %v", err)
		}
	})
}

func TestLedgerServiceRecordPaymentSucceeded(t *testing.T) {
	t.Run("books payment once and projects transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)
		outboxRepo := ledgerrepos.NewMockLedgerOutboxEventsRepository(ctrl)

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		txRepos.EXPECT().LedgerOutboxEventsRepository().Return(outboxRepo)

		accountRepo.EXPECT().Load(gomock.Any(), "ledger:clearing:provider:stripe").Return(nil, nil)
		accountRepo.EXPECT().Load(gomock.Any(), "wallet:available").Return(nil, nil)
		accountRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(2)
		outboxRepo.EXPECT().
			Append(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, evt eventpkg.Event) error {
				payload, ok := evt.EventData.(*ledgerprojection.LedgerTransactionProjected)
				if !ok {
					t.Fatalf("unexpected outbox payload type: %T", evt.EventData)
				}
				if payload.ReferenceType != "payment.succeeded" {
					t.Fatalf("unexpected reference type: %s", payload.ReferenceType)
				}
				if payload.ReferenceID != "pay-1" {
					t.Fatalf("unexpected reference id: %s", payload.ReferenceID)
				}
				return nil
			})

		service := NewLedgerService(baseRepo)
		err := service.RecordPaymentSucceeded(context.Background(), RecordPaymentSucceededCommand{
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("treats duplicate payment delivery as idempotent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)

		debitAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("ledger:clearing:provider:stripe")
		_, _ = debitAgg.BookPayment("payment:pay-1:succeeded", "pay-1", "wallet:available", "VND", -100, gomockTime())
		debitAgg.Root().Update()

		creditAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
		_, _ = creditAgg.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "VND", 100, gomockTime())
		creditAgg.Root().Update()

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "ledger:clearing:provider:stripe").Return(debitAgg, nil)
		accountRepo.EXPECT().Load(gomock.Any(), "wallet:available").Return(creditAgg, nil)

		service := NewLedgerService(baseRepo)
		err := service.RecordPaymentSucceeded(context.Background(), RecordPaymentSucceededCommand{
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
		})
		if err != nil {
			t.Fatalf("expected duplicate delivery to be idempotent, got %v", err)
		}
	})
}

func gomockTime() (out time.Time) {
	return time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
}
