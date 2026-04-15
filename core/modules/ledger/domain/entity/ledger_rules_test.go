package entity

import (
	"errors"
	"testing"
	"time"
)

func TestNewLedgerTransactionBuildsEntries(t *testing.T) {
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)

	transaction, err := NewLedgerTransaction(" txn-1 ", []LedgerEntryInput{
		{AccountID: " debit ", Currency: " usd ", Amount: -100},
		{AccountID: " credit ", Currency: "USD", Amount: 100},
	}, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if transaction.TransactionID != "txn-1" {
		t.Fatalf("unexpected transaction id: %s", transaction.TransactionID)
	}
	if len(transaction.Entries) != 2 {
		t.Fatalf("unexpected entries count: %d", len(transaction.Entries))
	}
	if transaction.Currency != "USD" {
		t.Fatalf("unexpected transaction currency: %s", transaction.Currency)
	}
	if transaction.Entries[0].AccountID != "debit" || transaction.Entries[1].AccountID != "credit" {
		t.Fatalf("unexpected account ids: %+v", transaction.Entries)
	}
	if transaction.Entries[0].Currency != "USD" || transaction.Entries[1].Currency != "USD" {
		t.Fatalf("unexpected entry currencies: %+v", transaction.Entries)
	}
}

func TestNewLedgerTransactionRejectsUnbalancedEntries(t *testing.T) {
	_, err := NewLedgerTransaction("txn-1", []LedgerEntryInput{
		{AccountID: "debit", Currency: "VND", Amount: -100},
		{AccountID: "credit", Currency: "VND", Amount: 90},
	}, time.Now().UTC())
	if !errors.Is(err, ErrLedgerEntriesUnbalanced) {
		t.Fatalf("expected unbalanced error, got %v", err)
	}
}

func TestNewPaymentSucceededBookingBuildsDomainBooking(t *testing.T) {
	booking, err := NewPaymentSucceededBooking(PaymentSucceededBookingInput{
		TransactionID:      "txn-1",
		ClearingAccountKey: "provider:stripe",
		CreditAccountID:    "credit",
		Currency:           "VND",
		Amount:             250,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if booking.PaymentID != "txn-1" {
		t.Fatalf("unexpected payment id: %s", booking.PaymentID)
	}
	if booking.LedgerTransactionID() != "payment:txn-1:succeeded" {
		t.Fatalf("unexpected ledger transaction id: %s", booking.LedgerTransactionID())
	}
	if booking.DebitAccountID != "ledger:clearing:provider:stripe" {
		t.Fatalf("unexpected debit account id: %s", booking.DebitAccountID)
	}

	entries := booking.LedgerEntries()
	if len(entries) != 2 || entries[0].Amount != -250 || entries[1].Amount != 250 {
		t.Fatalf("unexpected booking entries: %+v", entries)
	}
	if entries[0].Currency != "VND" || entries[1].Currency != "VND" {
		t.Fatalf("unexpected booking currencies: %+v", entries)
	}
}
