package aggregate

import (
	"errors"
	"testing"
	"time"

	valueobject "wechat-clone/core/modules/ledger/domain/value_object"
	sharedevents "wechat-clone/core/shared/contracts/events"
	eventpkg "wechat-clone/core/shared/pkg/event"
)

func TestLedgerAccountAggregateTransferLifecycle(t *testing.T) {
	aggregate, err := NewLedgerAccountAggregate("acc-1")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}

	applied, err := aggregate.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "vnd", 300, time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("BookPayment() error = %v", err)
	}
	if !applied {
		t.Fatalf("expected payment posting to apply")
	}
	if aggregate.Balance("VND") != 300 {
		t.Fatalf("expected balance 300, got %d", aggregate.Balance("VND"))
	}
	if aggregate.Root().Version() != 1 {
		t.Fatalf("expected version 1, got %d", aggregate.Root().Version())
	}
	if _, ok := aggregate.Root().Events()[0].EventData.(*EventLedgerAccountDepositFromIntent); !ok {
		t.Fatalf("expected first event to be EventLedgerAccountDepositFromIntent, got %T", aggregate.Root().Events()[0].EventData)
	}

	applied, err = aggregate.TransferToAccount("ledger-tx-1", "acc-2", "VND", 100, time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("TransferToAccount() error = %v", err)
	}
	if !applied {
		t.Fatalf("expected transfer posting to apply")
	}
	if aggregate.Balance("VND") != 200 {
		t.Fatalf("expected balance 200, got %d", aggregate.Balance("VND"))
	}
	if aggregate.Root().Version() != 2 {
		t.Fatalf("expected version 2, got %d", aggregate.Root().Version())
	}
}

func TestLedgerAccountAggregateRejectsOverspend(t *testing.T) {
	aggregate, err := NewLedgerAccountAggregate("acc-1")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}

	_, err = aggregate.TransferToAccount("ledger-tx-1", "acc-2", "USD", 100, time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrLedgerAccountInsufficientFunds) {
		t.Fatalf("expected insufficient funds error, got %v", err)
	}
}

func TestLedgerAccountAggregateReversePaymentLifecycle(t *testing.T) {
	aggregate, err := NewLedgerAccountAggregate("wallet:available")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}

	applied, err := aggregate.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "VND", 300, time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("BookPayment() error = %v", err)
	}
	if !applied {
		t.Fatalf("expected payment posting to apply")
	}

	applied, err = aggregate.ReversePayment("payment:pay-1:refunded", sharedevents.EventPaymentRefunded, "pay-1", "ledger:clearing:provider:stripe", "VND", -300, time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ReversePayment() error = %v", err)
	}
	if !applied {
		t.Fatalf("expected reversal posting to apply")
	}
	if aggregate.Balance("VND") != 0 {
		t.Fatalf("expected balance 0, got %d", aggregate.Balance("VND"))
	}
	if _, ok := aggregate.Root().Events()[1].EventData.(*EventLedgerAccountWithdrawFromRefund); !ok {
		t.Fatalf("expected refund event to be EventLedgerAccountWithdrawFromRefund, got %T", aggregate.Root().Events()[1].EventData)
	}

	applied, err = aggregate.ReversePayment("payment:pay-1:refunded", sharedevents.EventPaymentRefunded, "pay-1", "ledger:clearing:provider:stripe", "VND", -300, time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ReversePayment() duplicate error = %v", err)
	}
	if applied {
		t.Fatalf("expected duplicate reversal posting to be idempotent")
	}
}

func TestLedgerAccountAggregateDuplicatePostingIgnoresBookedAtForIdempotency(t *testing.T) {
	aggregate, err := NewLedgerAccountAggregate("wallet:available")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}

	firstBookedAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	applied, err := aggregate.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "VND", 300, firstBookedAt)
	if err != nil {
		t.Fatalf("BookPayment() error = %v", err)
	}
	if !applied {
		t.Fatalf("expected first payment posting to apply")
	}

	applied, err = aggregate.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "VND", 300, firstBookedAt.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("expected duplicate payment with different booked_at to remain idempotent, got %v", err)
	}
	if applied {
		t.Fatal("expected duplicate payment posting not to append a second posting")
	}
}

func TestLedgerAccountAggregateRejectsSelfTransfer(t *testing.T) {
	aggregate, err := NewLedgerAccountAggregate("acc-1")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}

	_, err = aggregate.TransferToAccount("ledger-tx-1", "acc-1", "VND", 100, time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrLedgerAccountAccountsMustDiffer) {
		t.Fatalf("expected accounts-must-differ error, got %v", err)
	}
}

func TestLedgerAccountAggregateLoadFromHistoryRejectsInvalidPostingPayload(t *testing.T) {
	aggregate, err := NewLedgerAccountAggregate("acc-1")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}

	err = aggregate.Root().LoadFromHistory(aggregate, []eventpkg.Event{
		{
			AggregateID: "acc-1",
			Version:     1,
			EventData: &EventLedgerAccountReceivedTransfer{
				TransactionID: "ledger-tx-1",
				FromAccountID: "acc-2",
				Currency:      "VND",
				Amount:        100,
			},
		},
	})
	if !errors.Is(err, ErrLedgerAccountBookedAtRequired) {
		t.Fatalf("expected booked_at required error, got %v", err)
	}
	if aggregate.Balance("VND") != 0 {
		t.Fatalf("expected invalid replay not to mutate balance, got %d", aggregate.Balance("VND"))
	}
}

func TestSameLedgerAccountPostingIgnoresBookedAt(t *testing.T) {
	left, err := NewLedgerAccountPaymentPosting(
		valueobject.LedgerAccountPostingInput{
			AccountID:             "wallet:available",
			TransactionID:         "payment:pay-1:succeeded",
			ReferenceType:         EventNameLedgerAccountDepositFromIntent,
			ReferenceID:           "pay-1",
			CounterpartyAccountID: "ledger:clearing:provider:stripe",
			Currency:              "VND",
			AmountDelta:           300,
			BookedAt:              time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
		},
	)
	if err != nil {
		t.Fatalf("NewLedgerAccountPaymentPosting() error = %v", err)
	}

	right, err := NewLedgerAccountPaymentPosting(
		valueobject.LedgerAccountPostingInput{
			AccountID:             "wallet:available",
			TransactionID:         "payment:pay-1:succeeded",
			ReferenceType:         EventNameLedgerAccountDepositFromIntent,
			ReferenceID:           "pay-1",
			CounterpartyAccountID: "ledger:clearing:provider:stripe",
			Currency:              "VND",
			AmountDelta:           300,
			BookedAt:              time.Date(2026, 4, 16, 10, 5, 0, 0, time.UTC),
		},
	)
	if err != nil {
		t.Fatalf("NewLedgerAccountPaymentPosting() error = %v", err)
	}

	if !SameLedgerAccountPosting(left, right) {
		t.Fatal("expected equivalent postings to remain idempotent when only booked_at differs")
	}
}
