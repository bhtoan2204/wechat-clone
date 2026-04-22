package aggregate

import (
	"testing"
	"time"

	"wechat-clone/core/modules/payment/domain/entity"
	sharedevents "wechat-clone/core/shared/contracts/events"
)

func TestNewProviderTopUpAggregateQueuesCreatedEvent(t *testing.T) {
	now := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	agg, err := NewProviderTopUpAggregate("txn-1", "stripe", 100, "VND", "wallet:available", map[string]string{"source": "test"}, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if agg.TransactionID() != "txn-1" {
		t.Fatalf("unexpected transaction id: %s", agg.TransactionID())
	}
	if agg.Status() != entity.PaymentStatusCreating {
		t.Fatalf("unexpected status: %s", agg.Status())
	}
	outbox := agg.PendingOutboxEvents()
	if len(outbox) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(outbox))
	}
	if outbox[0].EventName != sharedevents.EventPaymentCreated {
		t.Fatalf("unexpected event name: %s", outbox[0].EventName)
	}
	if outbox[0].AggregateID != "txn-1" {
		t.Fatalf("unexpected aggregate id: %s", outbox[0].AggregateID)
	}
	if outbox[0].AggregateType != AggregateTypePaymentIntent {
		t.Fatalf("unexpected aggregate type: %s", outbox[0].AggregateType)
	}
	if outbox[0].Version != 1 {
		t.Fatalf("unexpected aggregate version: %d", outbox[0].Version)
	}
	if outbox[0].CreatedAt != now.Unix() {
		t.Fatalf("unexpected envelope created at: %d", outbox[0].CreatedAt)
	}
	payload, ok := outbox[0].EventData.(sharedevents.PaymentCreatedEvent)
	if !ok {
		t.Fatalf("unexpected payload type: %T", outbox[0].EventData)
	}
	if payload.Metadata["source"] != "test" {
		t.Fatalf("unexpected payload metadata: %#v", payload.Metadata)
	}
}

func TestPaymentIntentAggregateApplySuccessQueuesProcessedAndOutbox(t *testing.T) {
	intent, err := entity.NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "wallet:available", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	agg, err := RestorePaymentIntentAggregate(intent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mutation, err := agg.ApplyProviderOutcome(entity.PaymentProviderResult{
		EventID:     "evt-1",
		EventType:   "checkout.session.completed",
		Status:      entity.PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
		ExternalRef: "cs-1",
	}, "", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mutation.Duplicate {
		t.Fatalf("expected first success not to be duplicate")
	}
	if !mutation.Persist {
		t.Fatalf("expected success to require persistence")
	}
	if agg.Status() != entity.PaymentStatusSuccess {
		t.Fatalf("unexpected status: %s", agg.Status())
	}
	processed := agg.PendingProcessedEvents()
	if len(processed) != 1 {
		t.Fatalf("expected 1 processed event, got %d", len(processed))
	}
	if processed[0].IdempotencyKey != sharedevents.EventPaymentSucceeded+":txn-1" {
		t.Fatalf("unexpected processed event idempotency key: %s", processed[0].IdempotencyKey)
	}
	outbox := agg.PendingOutboxEvents()
	if len(outbox) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(outbox))
	}
	if outbox[0].EventName != sharedevents.EventPaymentSucceeded {
		t.Fatalf("unexpected event name: %s", outbox[0].EventName)
	}
	if outbox[0].AggregateType != AggregateTypePaymentIntent {
		t.Fatalf("unexpected aggregate type: %s", outbox[0].AggregateType)
	}
	if outbox[0].Version != 1 {
		t.Fatalf("unexpected outbox version: %d", outbox[0].Version)
	}
	successPayload, ok := outbox[0].EventData.(sharedevents.PaymentSucceededEvent)
	if !ok {
		t.Fatalf("unexpected success payload type: %T", outbox[0].EventData)
	}
	if successPayload.ProviderPaymentRef != "cs-1" {
		t.Fatalf("unexpected success provider payment ref: %s", successPayload.ProviderPaymentRef)
	}
}

func TestPaymentIntentAggregateAssignsSequentialEnvelopeVersions(t *testing.T) {
	intent, err := entity.NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "wallet:available", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	agg, err := RestorePaymentIntentAggregate(intent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mutation, err := agg.ApplyProviderOutcome(entity.PaymentProviderResult{
		EventID:     "evt-1",
		EventType:   "checkout.session.completed",
		Status:      entity.PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
		ExternalRef: "cs-1",
	}, "https://checkout.test/session", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mutation.Duplicate || !mutation.Persist {
		t.Fatalf("expected success mutation to persist without duplicate, got %+v", mutation)
	}

	outbox := agg.PendingOutboxEvents()
	if len(outbox) != 2 {
		t.Fatalf("expected 2 outbox events, got %d", len(outbox))
	}
	if outbox[0].EventName != sharedevents.EventPaymentSucceeded {
		t.Fatalf("unexpected first event name: %s", outbox[0].EventName)
	}
	if outbox[1].EventName != sharedevents.EventPaymentCheckoutSessionCreated {
		t.Fatalf("unexpected second event name: %s", outbox[1].EventName)
	}
	if outbox[0].Version != 1 || outbox[1].Version != 2 {
		t.Fatalf("unexpected envelope versions: %d, %d", outbox[0].Version, outbox[1].Version)
	}
}

func TestPaymentIntentAggregateRestoreWithVersionContinuesEnvelopeSequence(t *testing.T) {
	intent, err := entity.NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "wallet:available", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	intent.Status = entity.PaymentStatusPending

	agg, err := RestorePaymentIntentAggregateWithVersion(intent, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mutation, err := agg.ApplyProviderOutcome(entity.PaymentProviderResult{
		EventID:     "evt-1",
		EventType:   "checkout.session.completed",
		Status:      entity.PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
		ExternalRef: "cs-1",
	}, "", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mutation.Duplicate || !mutation.Persist {
		t.Fatalf("expected success mutation to persist without duplicate, got %+v", mutation)
	}

	outbox := agg.PendingOutboxEvents()
	if len(outbox) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(outbox))
	}
	if outbox[0].Version != 3 {
		t.Fatalf("expected restored aggregate to continue at version 3, got %d", outbox[0].Version)
	}
}

func TestPaymentIntentAggregateIgnoresLateFailureAfterSuccess(t *testing.T) {
	intent, err := entity.NewProviderTopUpIntent("txn-1", "stripe", 100, "VND", "wallet:available", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	agg, err := RestorePaymentIntentAggregate(intent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = agg.ApplyProviderOutcome(entity.PaymentProviderResult{
		Status:      entity.PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
		ExternalRef: "cs-1",
	}, "", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	agg.MarkPersisted()

	mutation, err := agg.ApplyProviderOutcome(entity.PaymentProviderResult{
		Status:      entity.PaymentStatusFailed,
		Amount:      100,
		Currency:    "VND",
		ExternalRef: "cs-1",
	}, "", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !mutation.Duplicate {
		t.Fatalf("expected late failure to be duplicate/ignored")
	}
	if mutation.Persist {
		t.Fatalf("expected late failure not to require persistence")
	}
	if agg.Status() != entity.PaymentStatusSuccess {
		t.Fatalf("unexpected status: %s", agg.Status())
	}
}
