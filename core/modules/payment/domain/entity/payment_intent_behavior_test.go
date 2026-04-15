package entity

import (
	"errors"
	"testing"
	"time"

	sharedevents "go-socket/core/shared/contracts/events"
)

func TestNewPaymentIntentNormalizesFields(t *testing.T) {
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)

	intent, err := NewProviderTopUpIntent(" txn-1 ", " STRIPE ", 100, " vnd ", " credit ", now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if intent.TransactionID != "txn-1" {
		t.Fatalf("unexpected transaction id: %s", intent.TransactionID)
	}
	if intent.Provider != "stripe" {
		t.Fatalf("unexpected provider: %s", intent.Provider)
	}
	if intent.Currency != "VND" {
		t.Fatalf("unexpected currency: %s", intent.Currency)
	}
	if intent.ClearingAccountKey != "provider:stripe" {
		t.Fatalf("unexpected clearing account key: %s", intent.ClearingAccountKey)
	}
	if intent.CreditAccountID != "credit" {
		t.Fatalf("unexpected credit account: %s", intent.CreditAccountID)
	}
	if intent.Status != PaymentStatusCreating {
		t.Fatalf("unexpected status: %s", intent.Status)
	}
}

func TestNewPaymentIntentRejectsMissingClearingAccountKey(t *testing.T) {
	_, err := newPaymentIntent("txn-1", "stripe", 100, "VND", "", "credit", time.Now().UTC())
	if !errors.Is(err, ErrPaymentClearingAccountKeyMissing) {
		t.Fatalf("expected missing clearing account key error, got %v", err)
	}
}

func TestPaymentIntentProviderBehaviors(t *testing.T) {
	intent, err := NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "credit", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := intent.SetProviderState(" ext-1 ", "success", time.Now().UTC()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if intent.Status != PaymentStatusSuccess {
		t.Fatalf("unexpected status: %s", intent.Status)
	}
	if intent.ExternalRef != "ext-1" {
		t.Fatalf("unexpected external ref: %s", intent.ExternalRef)
	}

	if err := intent.ValidateProviderResult(999, "VND"); !errors.Is(err, ErrPaymentProviderAmountMismatch) {
		t.Fatalf("expected amount mismatch error, got %v", err)
	}
	if err := intent.ValidateProviderResult(100, "usd"); !errors.Is(err, ErrPaymentProviderCurrencyMismatch) {
		t.Fatalf("expected currency mismatch error, got %v", err)
	}

	if key := intent.PaymentIdempotencyKey("evt-1", ""); key != "evt-1" {
		t.Fatalf("unexpected idempotency key from event id: %s", key)
	}
	if key := intent.PaymentIdempotencyKey("", ""); key != "ext-1" {
		t.Fatalf("unexpected idempotency key from external ref: %s", key)
	}
}

func TestPaymentIntentApplyProviderResult(t *testing.T) {
	intent, err := NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "credit", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = intent.ApplyProviderResult(PaymentProviderResult{
		ExternalRef: " ext-2 ",
		Status:      "success",
		Amount:      100,
		Currency:    "vnd",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if intent.ExternalRef != "ext-2" {
		t.Fatalf("unexpected external ref: %s", intent.ExternalRef)
	}
	if intent.Status != PaymentStatusSuccess {
		t.Fatalf("unexpected status: %s", intent.Status)
	}
}

func TestPaymentIntentApplyProviderResultRestoresClearingAccountKey(t *testing.T) {
	intent := &PaymentIntent{
		TransactionID:      "txn-1",
		Provider:           "stripe",
		Amount:             100,
		Currency:           "VND",
		ClearingAccountKey: "",
		CreditAccountID:    "credit",
		Status:             PaymentStatusPending,
	}

	err := intent.ApplyProviderResult(PaymentProviderResult{
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if intent.ClearingAccountKey != "provider:stripe" {
		t.Fatalf("unexpected clearing account key: %s", intent.ClearingAccountKey)
	}
}

func TestPaymentIntentBuildsDomainEvents(t *testing.T) {
	now := time.Date(2026, 4, 7, 8, 0, 0, 0, time.UTC)
	intent, err := NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "credit", now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := intent.ApplyProviderResult(PaymentProviderResult{
		EventID:     "evt-1",
		EventType:   "payment.succeeded",
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, now); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	createdEvent := intent.CreatedEvent(map[string]string{"source": "test"}, now)
	if createdEvent.EventName != sharedevents.EventPaymentCreated {
		t.Fatalf("unexpected created event name: %s", createdEvent.EventName)
	}

	succeededEvent := intent.SucceededEvent(PaymentProviderResult{
		EventID:     "evt-1",
		EventType:   "payment.succeeded",
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
	}, now)
	if succeededEvent.EventName != sharedevents.EventPaymentSucceeded {
		t.Fatalf("unexpected success event name: %s", succeededEvent.EventName)
	}

	processedEvent, err := intent.NewProcessedEvent(PaymentProviderResult{
		EventID:     "evt-1",
		ExternalRef: "ref-1",
	}, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if processedEvent.IdempotencyKey != "evt-1" {
		t.Fatalf("unexpected idempotency key: %s", processedEvent.IdempotencyKey)
	}
}
