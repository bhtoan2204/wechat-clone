package aggregate

import (
	"errors"
	"time"

	paymententity "wechat-clone/core/modules/payment/domain/entity"
	sharedevents "wechat-clone/core/shared/contracts/events"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

var ErrPaymentIntentOccurredAtRequired = errors.New("occurred_at is required")

var AggregateTypePaymentIntent = eventpkg.AggregateTypeName((*PaymentIntentAggregate)(nil))

type PaymentIntentMutation struct {
	Duplicate bool
	Persist   bool
}

type PaymentIntentAggregate struct {
	intent          *paymententity.PaymentIntent
	processedEvents []*paymententity.ProcessedPaymentEvent
	outboxEvents    []eventpkg.Event
	version         int
}

func NewProviderTopUpAggregate(
	transactionID,
	provider string,
	amount int64,
	currency,
	creditAccountID string,
	metadata map[string]string,
	now time.Time,
) (*PaymentIntentAggregate, error) {
	now, err := normalizePaymentIntentOccurredAt(now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	intent, err := paymententity.NewProviderTopUpIntent(
		transactionID,
		provider,
		amount,
		currency,
		creditAccountID,
		now,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg := &PaymentIntentAggregate{intent: intent}
	agg.recordOutboxEvent(sharedevents.EventPaymentCreated, intent.BuildCreatedEventData(metadata, now), now)
	return agg, nil
}

func RestorePaymentIntentAggregate(intent *paymententity.PaymentIntent) (*PaymentIntentAggregate, error) {
	return RestorePaymentIntentAggregateWithVersion(intent, 0)
}

func RestorePaymentIntentAggregateWithVersion(intent *paymententity.PaymentIntent, version int) (*PaymentIntentAggregate, error) {
	if intent == nil {
		return nil, stackErr.Error(paymententity.ErrPaymentTransactionIDRequired)
	}
	clone := *intent
	if err := clone.ApplyProviderResult(clone.CurrentProviderResult(paymententity.PaymentProviderResult{}), clone.UpdatedAt); err != nil {
		return nil, stackErr.Error(err)
	}
	clone.UpdatedAt = intent.UpdatedAt.UTC()
	clone.CreatedAt = intent.CreatedAt.UTC()
	if version < 0 {
		version = 0
	}
	return &PaymentIntentAggregate{
		intent:       &clone,
		version:      version,
		outboxEvents: nil,
	}, nil
}

func (a *PaymentIntentAggregate) Snapshot() *paymententity.PaymentIntent {
	if a == nil || a.intent == nil {
		return nil
	}
	clone := *a.intent
	return &clone
}

func (a *PaymentIntentAggregate) TransactionID() string {
	if a == nil || a.intent == nil {
		return ""
	}
	return a.intent.TransactionID
}

func (a *PaymentIntentAggregate) Provider() string {
	if a == nil || a.intent == nil {
		return ""
	}
	return a.intent.Provider
}

func (a *PaymentIntentAggregate) ExternalRef() string {
	if a == nil || a.intent == nil {
		return ""
	}
	return a.intent.ExternalRef
}

func (a *PaymentIntentAggregate) Status() string {
	if a == nil || a.intent == nil {
		return ""
	}
	return a.intent.Status
}

func (a *PaymentIntentAggregate) Version() int {
	if a == nil {
		return 0
	}
	return a.version
}

func (a *PaymentIntentAggregate) PendingProcessedEvents() []*paymententity.ProcessedPaymentEvent {
	if len(a.processedEvents) == 0 {
		return nil
	}
	items := make([]*paymententity.ProcessedPaymentEvent, 0, len(a.processedEvents))
	for _, item := range a.processedEvents {
		if item == nil {
			continue
		}
		clone := *item
		items = append(items, &clone)
	}
	return items
}

func (a *PaymentIntentAggregate) PendingOutboxEvents() []eventpkg.Event {
	if len(a.outboxEvents) == 0 {
		return nil
	}
	items := make([]eventpkg.Event, len(a.outboxEvents))
	copy(items, a.outboxEvents)
	return items
}

func (a *PaymentIntentAggregate) MarkPersisted() {
	if a == nil {
		return
	}
	a.processedEvents = nil
	a.outboxEvents = nil
}

func (a *PaymentIntentAggregate) ApplyProviderOutcome(
	result paymententity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
	occurredAt time.Time,
) (PaymentIntentMutation, error) {
	if a == nil || a.intent == nil {
		return PaymentIntentMutation{}, stackErr.Error(paymententity.ErrPaymentTransactionIDRequired)
	}
	occurredAt, err := normalizePaymentIntentOccurredAt(occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}

	switch paymententity.NormalizePaymentStatus(result.Status) {
	case paymententity.PaymentStatusSuccess:
		return a.applySuccessfulOutcome(result, checkoutURL, emitCheckoutEvent, occurredAt)
	case paymententity.PaymentStatusRefunded, paymententity.PaymentStatusChargeback:
		return a.applyReversedOutcome(result, checkoutURL, emitCheckoutEvent, occurredAt)
	default:
		return a.applyNonFinalOutcome(result, checkoutURL, emitCheckoutEvent, occurredAt)
	}
}

func (a *PaymentIntentAggregate) MarkCreateFailed(occurredAt time.Time) (PaymentIntentMutation, error) {
	if a == nil || a.intent == nil {
		return PaymentIntentMutation{}, stackErr.Error(paymententity.ErrPaymentTransactionIDRequired)
	}
	occurredAt, err := normalizePaymentIntentOccurredAt(occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}

	transition, err := a.intent.MarkCreateFailed(occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || !transition.StateChanged {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	failedResult := a.intent.CurrentProviderResult(paymententity.PaymentProviderResult{Status: paymententity.PaymentStatusFailed})
	a.recordOutboxEvent(sharedevents.EventPaymentFailed, a.intent.BuildFailedEventData(failedResult, a.intent.UpdatedAt), a.intent.UpdatedAt)
	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) applyNonFinalOutcome(
	result paymententity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
	occurredAt time.Time,
) (PaymentIntentMutation, error) {
	transition, err := a.intent.TransitionProviderResult(result, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || (!transition.StateChanged && !transition.ExternalRefChanged) {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt)
	if transition.Type == paymententity.PaymentTransitionFailed {
		a.recordOutboxEvent(
			sharedevents.EventPaymentFailed,
			a.intent.BuildFailedEventData(a.intent.CurrentProviderResult(result), occurredAt),
			occurredAt,
		)
	}

	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) applySuccessfulOutcome(
	result paymententity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
	occurredAt time.Time,
) (PaymentIntentMutation, error) {
	transition, err := a.intent.TransitionProviderResult(result, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || transition.Type == paymententity.PaymentTransitionNone {
		if transition.ExternalRefChanged {
			a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt)
			return PaymentIntentMutation{Duplicate: true, Persist: true}, nil
		}
		return PaymentIntentMutation{Duplicate: true}, nil
	}
	if transition.Type != paymententity.PaymentTransitionSucceeded {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	processedEvent, err := a.intent.NewProcessedTransitionEvent(sharedevents.EventPaymentSucceeded, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}

	a.recordProcessedEvent(processedEvent)
	a.recordOutboxEvent(
		sharedevents.EventPaymentSucceeded,
		a.intent.BuildSucceededEventData(a.intent.CurrentProviderResult(result), occurredAt),
		occurredAt,
	)
	a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt)
	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) applyReversedOutcome(
	result paymententity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
	occurredAt time.Time,
) (PaymentIntentMutation, error) {
	transition, err := a.intent.TransitionProviderResult(result, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || transition.Type == paymententity.PaymentTransitionNone {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	var (
		processedEvent *paymententity.ProcessedPaymentEvent
		reversalData   interface{}
		reversalName   string
	)
	switch transition.Type {
	case paymententity.PaymentTransitionRefunded:
		processedEvent, err = a.intent.NewProcessedTransitionEvent(sharedevents.EventPaymentRefunded, occurredAt)
		if err != nil {
			return PaymentIntentMutation{}, stackErr.Error(err)
		}
		reversalName = sharedevents.EventPaymentRefunded
		reversalData = a.intent.BuildRefundedEventData(a.intent.CurrentProviderResult(result), occurredAt)
	case paymententity.PaymentTransitionChargeback:
		processedEvent, err = a.intent.NewProcessedTransitionEvent(sharedevents.EventPaymentChargeback, occurredAt)
		if err != nil {
			return PaymentIntentMutation{}, stackErr.Error(err)
		}
		reversalName = sharedevents.EventPaymentChargeback
		reversalData = a.intent.BuildChargebackEventData(a.intent.CurrentProviderResult(result), occurredAt)
	default:
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	a.recordProcessedEvent(processedEvent)
	a.recordOutboxEvent(reversalName, reversalData, occurredAt)
	a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt)
	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) recordProcessedEvent(evt *paymententity.ProcessedPaymentEvent) {
	if a == nil || evt == nil {
		return
	}
	clone := *evt
	a.processedEvents = append(a.processedEvents, &clone)
}

// Aggregate root owns the outbox envelope; entities only supply payload facts.
func (a *PaymentIntentAggregate) recordOutboxEvent(eventName string, eventData interface{}, occurredAt time.Time) {
	if a == nil || a.intent == nil {
		return
	}
	a.version++
	a.outboxEvents = append(a.outboxEvents, eventpkg.Event{
		AggregateID:   a.intent.TransactionID,
		AggregateType: AggregateTypePaymentIntent,
		Version:       a.version,
		EventName:     eventName,
		EventData:     eventData,
		CreatedAt:     occurredAt.Unix(),
	})
}

func (a *PaymentIntentAggregate) recordCheckoutSessionEvent(checkoutURL string, emitCheckoutEvent bool, occurredAt time.Time) {
	if a == nil || a.intent == nil {
		return
	}
	if !emitCheckoutEvent || !a.intent.ShouldEmitCheckoutSessionCreated(checkoutURL) {
		return
	}
	a.recordOutboxEvent(
		sharedevents.EventPaymentCheckoutSessionCreated,
		a.intent.BuildCheckoutSessionCreatedEventData(checkoutURL, occurredAt),
		occurredAt,
	)
}

func normalizePaymentIntentOccurredAt(value time.Time) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, ErrPaymentIntentOccurredAtRequired
	}
	return value.UTC(), nil
}
