package service

import (
	"context"
	"errors"
	"testing"
	"time"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	valueobject "wechat-clone/core/modules/ledger/domain/value_object"
	sharedevents "wechat-clone/core/shared/contracts/events"

	"go.uber.org/mock/gomock"
)

func TestLedgerServiceTransferToAccount(t *testing.T) {
	t.Run("posts transfer across account streams and outbox", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)

		fromAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("acc-from")
		fromAgg.Balances["VND"] = 500

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "acc-from").Return(fromAgg, nil).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "acc-to").Return(nil, nil).AnyTimes()
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
		accountRepo.EXPECT().Load(gomock.Any(), "acc-from").Return(fromAgg, nil).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "acc-to").Return(nil, nil).AnyTimes()

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

	t.Run("treats duplicate transfer as idempotent before balance checks", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)
		fromPosting, err := ledgeraggregate.NewLedgerAccountTransferOutPosting(valueobject.LedgerAccountTransferPostingInput{
			AccountID:             "acc-from",
			TransactionID:         "ledger-tx-3",
			CounterpartyAccountID: "acc-to",
			Currency:              "USD",
			Amount:                100,
			BookedAt:              gomockTime(),
		})
		if err != nil {
			t.Fatalf("NewLedgerAccountTransferOutPosting() error = %v", err)
		}
		toPosting, err := ledgeraggregate.NewLedgerAccountTransferInPosting(valueobject.LedgerAccountTransferPostingInput{
			AccountID:             "acc-to",
			TransactionID:         "ledger-tx-3",
			CounterpartyAccountID: "acc-from",
			Currency:              "USD",
			Amount:                100,
			BookedAt:              gomockTime(),
		})
		if err != nil {
			t.Fatalf("NewLedgerAccountTransferInPosting() error = %v", err)
		}
		fromAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("acc-from")
		fromAgg.PostedTransactions[fromPosting.TransactionID] = fromPosting
		toAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("acc-to")
		toAgg.PostedTransactions[toPosting.TransactionID] = toPosting

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "acc-from").Return(fromAgg, nil)
		accountRepo.EXPECT().Load(gomock.Any(), "acc-to").Return(toAgg, nil)

		service := NewLedgerService(baseRepo)
		_, err = service.TransferToAccount(context.Background(), TransferToAccountCommand{
			TransactionID: "ledger-tx-3",
			FromAccountID: "acc-from",
			ToAccountID:   "acc-to",
			Currency:      "USD",
			Amount:        100,
		})
		if err != nil {
			t.Fatalf("expected duplicate transfer to be idempotent, got %v", err)
		}
	})
}

func TestLedgerServiceRecordPaymentSucceeded(t *testing.T) {
	t.Run("books payment once and projects transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)
		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "ledger:clearing:provider:stripe").Return(nil, nil).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "wallet:available").Return(nil, nil).AnyTimes()
		accountRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(2)
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
		debitPosting, err := ledgeraggregate.NewLedgerAccountPaymentPosting(valueobject.LedgerAccountPostingInput{
			AccountID:             "ledger:clearing:provider:stripe",
			TransactionID:         "payment:pay-1:succeeded",
			ReferenceType:         ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent,
			ReferenceID:           "pay-1",
			CounterpartyAccountID: "wallet:available",
			Currency:              "VND",
			AmountDelta:           -100,
			BookedAt:              gomockTime(),
		})
		if err != nil {
			t.Fatalf("NewLedgerAccountPaymentPosting() error = %v", err)
		}
		creditPosting, err := ledgeraggregate.NewLedgerAccountPaymentPosting(valueobject.LedgerAccountPostingInput{
			AccountID:             "wallet:available",
			TransactionID:         "payment:pay-1:succeeded",
			ReferenceType:         ledgeraggregate.EventNameLedgerAccountDepositFromIntent,
			ReferenceID:           "pay-1",
			CounterpartyAccountID: "ledger:clearing:provider:stripe",
			Currency:              "VND",
			AmountDelta:           100,
			BookedAt:              gomockTime(),
		})
		if err != nil {
			t.Fatalf("NewLedgerAccountPaymentPosting() error = %v", err)
		}
		debitAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("ledger:clearing:provider:stripe")
		debitAgg.PostedTransactions[debitPosting.TransactionID] = debitPosting
		creditAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
		creditAgg.PostedTransactions[creditPosting.TransactionID] = creditPosting

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "ledger:clearing:provider:stripe").Return(debitAgg, nil)
		accountRepo.EXPECT().Load(gomock.Any(), "wallet:available").Return(creditAgg, nil)

		service := NewLedgerService(baseRepo)
		err = service.RecordPaymentSucceeded(context.Background(), RecordPaymentSucceededCommand{
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

func TestLedgerServiceRecordPaymentReversed(t *testing.T) {
	t.Run("books refunded payment reversal and projects transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)
		walletAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
		_, _ = walletAgg.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "VND", 100, gomockTime())
		walletAgg.Root().Update()

		clearingAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("ledger:clearing:provider:stripe")
		_, _ = clearingAgg.BookPayment("payment:pay-1:succeeded", "pay-1", "wallet:available", "VND", -100, gomockTime())
		clearingAgg.Root().Update()

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "wallet:available").Return(walletAgg, nil).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "ledger:clearing:provider:stripe").Return(clearingAgg, nil).AnyTimes()
		accountRepo.EXPECT().
			Save(gomock.Any(), gomock.AssignableToTypeOf(&ledgeraggregate.LedgerAccountAggregate{})).
			DoAndReturn(func(_ context.Context, aggregate *ledgeraggregate.LedgerAccountAggregate) error {
				switch aggregate.AggregateID() {
				case "wallet:available":
					if aggregate.Balance("VND") != 0 {
						t.Fatalf("expected wallet balance 0, got %d", aggregate.Balance("VND"))
					}
				case "ledger:clearing:provider:stripe":
					if aggregate.Balance("VND") != 0 {
						t.Fatalf("expected clearing balance 0, got %d", aggregate.Balance("VND"))
					}
				default:
					t.Fatalf("unexpected aggregate saved: %s", aggregate.AggregateID())
				}
				return nil
			}).Times(2)
		service := NewLedgerService(baseRepo)
		err := service.RecordPaymentReversed(context.Background(), RecordPaymentReversedCommand{
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
			ReversalType:       sharedevents.EventPaymentRefunded,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("treats duplicate reversal delivery as idempotent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		baseRepo := ledgerrepos.NewMockRepos(ctrl)
		txRepos := ledgerrepos.NewMockRepos(ctrl)
		accountRepo := ledgerrepos.NewMockLedgerAccountAggregateRepository(ctrl)
		debitPosting, err := ledgeraggregate.NewLedgerAccountPaymentPosting(valueobject.LedgerAccountPostingInput{
			AccountID:             "wallet:available",
			TransactionID:         "payment:pay-1:refunded",
			ReferenceType:         ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund,
			ReferenceID:           "pay-1",
			CounterpartyAccountID: "ledger:clearing:provider:stripe",
			Currency:              "VND",
			AmountDelta:           -100,
			BookedAt:              gomockTime(),
		})
		if err != nil {
			t.Fatalf("NewLedgerAccountPaymentPosting() error = %v", err)
		}
		creditPosting, err := ledgeraggregate.NewLedgerAccountPaymentPosting(valueobject.LedgerAccountPostingInput{
			AccountID:             "ledger:clearing:provider:stripe",
			TransactionID:         "payment:pay-1:refunded",
			ReferenceType:         ledgeraggregate.EventNameLedgerAccountDepositFromRefund,
			ReferenceID:           "pay-1",
			CounterpartyAccountID: "wallet:available",
			Currency:              "VND",
			AmountDelta:           100,
			BookedAt:              gomockTime(),
		})
		if err != nil {
			t.Fatalf("NewLedgerAccountPaymentPosting() error = %v", err)
		}
		debitAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
		debitAgg.PostedTransactions[debitPosting.TransactionID] = debitPosting
		creditAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("ledger:clearing:provider:stripe")
		creditAgg.PostedTransactions[creditPosting.TransactionID] = creditPosting

		baseRepo.EXPECT().
			WithTransaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(ledgerrepos.Repos) error) error {
				return fn(txRepos)
			})
		txRepos.EXPECT().LedgerAccountAggregateRepository().Return(accountRepo).AnyTimes()
		accountRepo.EXPECT().Load(gomock.Any(), "wallet:available").Return(debitAgg, nil)
		accountRepo.EXPECT().Load(gomock.Any(), "ledger:clearing:provider:stripe").Return(creditAgg, nil)

		service := NewLedgerService(baseRepo)
		err = service.RecordPaymentReversed(context.Background(), RecordPaymentReversedCommand{
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
			ReversalType:       sharedevents.EventPaymentRefunded,
		})
		if err != nil {
			t.Fatalf("expected duplicate delivery to be idempotent, got %v", err)
		}
	})
}

func gomockTime() (out time.Time) {
	return time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
}
