package aggregate

import (
	"errors"
	"testing"
	"time"
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
