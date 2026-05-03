package service

import (
	"context"
	"strings"
	"testing"
	"time"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	ledgerentity "wechat-clone/core/modules/ledger/domain/entity"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	sharedevents "wechat-clone/core/shared/contracts/events"
	eventpkg "wechat-clone/core/shared/pkg/event"

	"go.uber.org/mock/gomock"
)

func TestLedgerServiceRecordPaymentReconciliationFailed(t *testing.T) {
	t.Run("appends failure event through configured outbox", func(t *testing.T) {
		outbox := &ledgerReconciliationOutboxFake{}
		service := NewLedgerService(ledgerReposWithOutboxFake{outbox: outbox})
		failedAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

		if err := service.RecordPaymentReconciliationFailed(context.Background(), RecordPaymentReconciliationFailedCommand{
			PaymentID:          "pay-1",
			TransactionID:      "tx-1",
			Provider:           "stripe",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
			FeeAmount:          5,
			ProviderAmount:     105,
			Reason:             "provider mismatch",
			FailedAt:           failedAt,
		}); err != nil {
			t.Fatalf("RecordPaymentReconciliationFailed() error = %v", err)
		}
		if outbox.appended.EventName != sharedevents.EventLedgerPaymentReconciliationFailed {
			t.Fatalf("unexpected event name: %s", outbox.appended.EventName)
		}
		payload, ok := outbox.appended.EventData.(sharedevents.LedgerPaymentReconciliationFailedEvent)
		if !ok {
			t.Fatalf("unexpected payload type: %T", outbox.appended.EventData)
		}
		if payload.PaymentID != "pay-1" || payload.Reason != "provider mismatch" || !payload.FailedAt.Equal(failedAt) {
			t.Fatalf("unexpected payload: %#v", payload)
		}
	})

	t.Run("requires publisher", func(t *testing.T) {
		service := NewLedgerService(ledgerReposWithoutOutboxFake{})

		err := service.RecordPaymentReconciliationFailed(context.Background(), RecordPaymentReconciliationFailedCommand{PaymentID: "pay-1"})
		if err == nil || !strings.Contains(err.Error(), "publisher is not configured") {
			t.Fatalf("expected publisher error, got %v", err)
		}
	})

	t.Run("requires payment id or transaction id", func(t *testing.T) {
		service := NewLedgerService(ledgerReposWithOutboxFake{outbox: &ledgerReconciliationOutboxFake{}})

		err := service.RecordPaymentReconciliationFailed(context.Background(), RecordPaymentReconciliationFailedCommand{})
		if err == nil || !strings.Contains(err.Error(), "payment_id is required") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})
}

func TestLedgerServiceHelperCoverage(t *testing.T) {
	service := NewLedgerService(nil)

	newAgg, err := service.loadLedgerAccount(context.Background(), ledgerReposStaticFake{loadResult: nil}, "acc-new")
	if err != nil {
		t.Fatalf("loadLedgerAccount() new aggregate error = %v", err)
	}
	if newAgg.AggregateID() != "acc-new" {
		t.Fatalf("unexpected new aggregate id: %s", newAgg.AggregateID())
	}

	existingAgg, _ := ledgeraggregate.NewLedgerAccountAggregate("acc-existing")
	loadedAgg, err := service.loadLedgerAccount(context.Background(), ledgerReposStaticFake{loadResult: existingAgg}, "acc-existing")
	if err != nil {
		t.Fatalf("loadLedgerAccount() existing aggregate error = %v", err)
	}
	if loadedAgg != existingAgg {
		t.Fatalf("expected existing aggregate to be returned")
	}

	loadedAccounts, err := service.loadLedgerAccounts(context.Background(), ledgerReposStaticFake{}, " acc-1 ", "", "acc-1", "acc-2")
	if err != nil {
		t.Fatalf("loadLedgerAccounts() error = %v", err)
	}
	if len(loadedAccounts) != 2 || loadedAccounts.account("acc-1") == nil || loadedAccounts.account("acc-2") == nil {
		t.Fatalf("unexpected loaded accounts: %#v", loadedAccounts)
	}
	if (loadedLedgerAccounts)(nil).account("acc-1") != nil {
		t.Fatalf("expected nil loaded accounts map to return nil")
	}

	if alreadyApplied, err := service.saveLedgerAccount(context.Background(), ledgerReposStaticFake{saveErr: ledgerrepos.ErrAlreadyApplied}, existingAgg); err != nil || !alreadyApplied {
		t.Fatalf("saveLedgerAccount() ErrAlreadyApplied = (%v, %v), want (true, nil)", alreadyApplied, err)
	}

	if debitLedgerEventNameForPaymentReversal(sharedevents.EventPaymentRefunded) != ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund {
		t.Fatalf("unexpected payment reversal debit refund event")
	}
	if debitLedgerEventNameForPaymentReversal(sharedevents.EventPaymentChargeback) != ledgeraggregate.EventNameLedgerAccountWithdrawFromChargeback {
		t.Fatalf("unexpected payment reversal debit chargeback event")
	}
	if debitLedgerEventNameForPaymentReversal("other") != "" {
		t.Fatalf("unexpected default payment reversal debit event")
	}
	if creditLedgerEventNameForPaymentReversal(sharedevents.EventPaymentRefunded) != ledgeraggregate.EventNameLedgerAccountDepositFromRefund {
		t.Fatalf("unexpected payment reversal credit refund event")
	}
	if creditLedgerEventNameForPaymentReversal(sharedevents.EventPaymentChargeback) != ledgeraggregate.EventNameLedgerAccountDepositFromChargeback {
		t.Fatalf("unexpected payment reversal credit chargeback event")
	}
	if creditLedgerEventNameForPaymentReversal("other") != "" {
		t.Fatalf("unexpected default payment reversal credit event")
	}
}

func TestLedgerServiceTransferToAccountWithFee(t *testing.T) {
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
	accountRepo.EXPECT().Load(gomock.Any(), "ledger:fees").Return(nil, nil).AnyTimes()
	accountRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(3)

	service := NewLedgerService(baseRepo)
	transaction, err := service.TransferToAccount(context.Background(), TransferToAccountCommand{
		TransactionID: "ledger-tx-fee",
		FromAccountID: "acc-from",
		ToAccountID:   "acc-to",
		Currency:      "VND",
		Amount:        100,
		FeeAmount:     5,
		FeeAccountID:  "ledger:fees",
	})
	if err != nil {
		t.Fatalf("TransferToAccount() error = %v", err)
	}
	if len(transaction.Entries) != 4 {
		t.Fatalf("expected 4 entries including fee, got %d", len(transaction.Entries))
	}
}

func TestLedgerPaymentEventsFromPostingsRejectsUnsupportedReference(t *testing.T) {
	_, err := ledgerPaymentEventsFromPostings([]ledgerPostingEventInput{{
		accountID: "acc-1",
		posting: ledgerentity.LedgerAccountPosting{
			TransactionID: "tx-1",
			ReferenceType: "unsupported",
			ReferenceID:   "ref-1",
			Currency:      "VND",
			AmountDelta:   100,
			BookedAt:      gomockTime(),
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "unsupported ledger posting") {
		t.Fatalf("expected unsupported posting error, got %v", err)
	}
}

type ledgerReconciliationOutboxFake struct {
	appended eventpkg.Event
}

func (f *ledgerReconciliationOutboxFake) Append(_ context.Context, event eventpkg.Event) error {
	f.appended = event
	return nil
}

type ledgerReposWithOutboxFake struct {
	outbox eventpkg.Store
}

func (r ledgerReposWithOutboxFake) LedgerAccountAggregateRepository() ledgerrepos.LedgerAccountAggregateRepository {
	return nil
}

func (r ledgerReposWithOutboxFake) WithTransaction(context.Context, func(ledgerrepos.Repos) error) error {
	return nil
}

func (r ledgerReposWithOutboxFake) PaymentReconciliationFailureEventStore() eventpkg.Store {
	return r.outbox
}

type ledgerReposWithoutOutboxFake struct{}

func (r ledgerReposWithoutOutboxFake) LedgerAccountAggregateRepository() ledgerrepos.LedgerAccountAggregateRepository {
	return nil
}

func (r ledgerReposWithoutOutboxFake) WithTransaction(context.Context, func(ledgerrepos.Repos) error) error {
	return nil
}

type ledgerReposStaticFake struct {
	loadResult *ledgeraggregate.LedgerAccountAggregate
	saveErr    error
}

func (r ledgerReposStaticFake) LedgerAccountAggregateRepository() ledgerrepos.LedgerAccountAggregateRepository {
	return ledgerAccountRepoStaticFake{loadResult: r.loadResult, saveErr: r.saveErr}
}

func (r ledgerReposStaticFake) WithTransaction(_ context.Context, fn func(ledgerrepos.Repos) error) error {
	return fn(r)
}

type ledgerAccountRepoStaticFake struct {
	loadResult *ledgeraggregate.LedgerAccountAggregate
	saveErr    error
}

func (r ledgerAccountRepoStaticFake) Load(context.Context, string) (*ledgeraggregate.LedgerAccountAggregate, error) {
	return r.loadResult, nil
}

func (r ledgerAccountRepoStaticFake) Save(context.Context, *ledgeraggregate.LedgerAccountAggregate) error {
	return r.saveErr
}
