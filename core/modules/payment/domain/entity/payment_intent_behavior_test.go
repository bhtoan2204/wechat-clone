package entity

import (
	"errors"
	"testing"
	"time"

	sharedevents "wechat-clone/core/shared/contracts/events"
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

func TestNewPaymentIntentDerivesClearingAccountKeyWhenMissing(t *testing.T) {
	intent, err := newPaymentIntent("txn-1", "stripe", 100, "VND", "", "credit", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if intent.ClearingAccountKey != "provider:stripe" {
		t.Fatalf("unexpected clearing account key: %s", intent.ClearingAccountKey)
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
	if key := intent.TransitionIdempotencyKey(sharedevents.EventPaymentSucceeded); key != sharedevents.EventPaymentSucceeded+":txn-1" {
		t.Fatalf("unexpected transition idempotency key: %s", key)
	}
}

func TestPaymentIntentTransitionIgnoresFailAfterSuccess(t *testing.T) {
	now := time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC)
	intent, err := NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "credit", now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	transition, err := intent.TransitionProviderResult(PaymentProviderResult{
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, now)
	if err != nil {
		t.Fatalf("expected success transition, got %v", err)
	}
	if transition.Type != PaymentTransitionSucceeded {
		t.Fatalf("unexpected transition type: %s", transition.Type)
	}

	transition, err = intent.TransitionProviderResult(PaymentProviderResult{
		ExternalRef: "ref-1",
		Status:      PaymentStatusFailed,
		Amount:      100,
		Currency:    "VND",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected ignored late failure, got %v", err)
	}
	if !transition.Ignored {
		t.Fatalf("expected late failure transition to be ignored")
	}
	if intent.Status != PaymentStatusSuccess {
		t.Fatalf("expected status to stay success, got %s", intent.Status)
	}
	if !intent.IsTerminal() {
		t.Fatalf("expected success to be terminal")
	}
}

func TestPaymentIntentTransitionAllowsSuccessThenRefunded(t *testing.T) {
	now := time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC)
	intent, err := NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "credit", now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := intent.ApplyProviderResult(PaymentProviderResult{
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, now); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := intent.ValidateProviderResultForStatus(PaymentStatusRefunded, 50, "VND"); err != nil {
		t.Fatalf("expected partial refund validation to pass, got %v", err)
	}

	transition, err := intent.TransitionProviderResult(PaymentProviderResult{
		EventID:     "evt-refund-1",
		EventType:   sharedevents.EventPaymentRefunded,
		ExternalRef: "ref-1",
		Status:      PaymentStatusRefunded,
		Amount:      100,
		Currency:    "VND",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if transition.Type != PaymentTransitionRefunded {
		t.Fatalf("unexpected transition type: %s", transition.Type)
	}
	if intent.Status != PaymentStatusRefunded {
		t.Fatalf("expected refunded status, got %s", intent.Status)
	}
	if !intent.IsRefunded() || !intent.IsTerminal() || !intent.IsFinalized() {
		t.Fatalf("expected refunded payment to be terminal and finalized")
	}
}

func TestPaymentIntentTransitionAllowsSuccessThenChargeback(t *testing.T) {
	now := time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC)
	intent, err := NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "credit", now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := intent.ApplyProviderResult(PaymentProviderResult{
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, now); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	transition, err := intent.TransitionProviderResult(PaymentProviderResult{
		EventID:     "evt-chargeback-1",
		EventType:   sharedevents.EventPaymentChargeback,
		ExternalRef: "ref-1",
		Status:      PaymentStatusChargeback,
		Amount:      100,
		Currency:    "VND",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if transition.Type != PaymentTransitionChargeback {
		t.Fatalf("unexpected transition type: %s", transition.Type)
	}
	if intent.Status != PaymentStatusChargeback {
		t.Fatalf("expected chargeback status, got %s", intent.Status)
	}
	if !intent.IsChargeback() || !intent.IsTerminal() || !intent.IsFinalized() {
		t.Fatalf("expected chargeback payment to be terminal and finalized")
	}
}

func TestPaymentIntentTerminalFailureIgnoresLateSuccess(t *testing.T) {
	now := time.Date(2026, 4, 17, 8, 0, 0, 0, time.UTC)
	intent, err := NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "credit", now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := intent.ApplyProviderResult(PaymentProviderResult{
		ExternalRef: "ref-1",
		Status:      PaymentStatusFailed,
		Amount:      100,
		Currency:    "VND",
	}, now); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	transition, err := intent.TransitionProviderResult(PaymentProviderResult{
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !transition.Ignored {
		t.Fatalf("expected late success to be ignored after failure")
	}
	if intent.Status != PaymentStatusFailed {
		t.Fatalf("expected status to stay failed, got %s", intent.Status)
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

func TestPaymentIntentBuildsEventData(t *testing.T) {
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

	createdEventData := intent.BuildCreatedEventData(map[string]string{"source": "test"}, now)
	if createdEventData.TransactionID != "txn-1" {
		t.Fatalf("unexpected created transaction id: %s", createdEventData.TransactionID)
	}
	if createdEventData.CreatedAt != now {
		t.Fatalf("unexpected created event time: %v", createdEventData.CreatedAt)
	}

	succeededEventData := intent.BuildSucceededEventData(PaymentProviderResult{
		EventID:     "evt-1",
		EventType:   "payment.succeeded",
		ExternalRef: "ref-1",
		Status:      PaymentStatusSuccess,
	}, now)
	if succeededEventData.IdempotencyKey != sharedevents.EventPaymentSucceeded+":txn-1" {
		t.Fatalf("unexpected success idempotency key: %s", succeededEventData.IdempotencyKey)
	}
	if succeededEventData.ProviderPaymentRef != "ref-1" {
		t.Fatalf("unexpected success provider payment ref: %s", succeededEventData.ProviderPaymentRef)
	}

	processedEvent, err := intent.NewProcessedTransitionEvent(sharedevents.EventPaymentSucceeded, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if processedEvent.IdempotencyKey != sharedevents.EventPaymentSucceeded+":txn-1" {
		t.Fatalf("unexpected processed transition idempotency key: %s", processedEvent.IdempotencyKey)
	}

	legacyProcessedEvent, err := intent.NewProcessedEvent(PaymentProviderResult{
		EventID:     "evt-1",
		ExternalRef: "ref-1",
	}, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if legacyProcessedEvent.IdempotencyKey != "evt-1" {
		t.Fatalf("unexpected event replay idempotency key: %s", legacyProcessedEvent.IdempotencyKey)
	}
}
