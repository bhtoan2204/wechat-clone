package aggregate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	paymententity "wechat-clone/core/modules/payment/domain/entity"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

var ErrPaymentIntentOccurredAtRequired = errors.New("occurred_at is required")

var AggregateTypePaymentIntent = event.AggregateTypeName((*PaymentIntentAggregate)(nil))

type PaymentIntentMutation struct {
	Duplicate bool
	Persist   bool
}

type PaymentIntentAggregate struct {
	event.AggregateRoot

	intent          *paymententity.PaymentIntent
	processedEvents []*paymententity.ProcessedPaymentEvent
}

func NewProviderTopUpAggregate(
	transactionID,
	provider string,
	amount int64,
	feeAmount int64,
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
		feeAmount,
		currency,
		creditAccountID,
		now,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := newPaymentIntentAggregate(intent)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.recordPaymentCreated(metadata, now); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func NewProviderWithdrawalAggregate(
	transactionID,
	provider string,
	amount int64,
	feeAmount int64,
	currency,
	destinationAccountID,
	debitAccountID string,
	metadata map[string]string,
	now time.Time,
) (*PaymentIntentAggregate, error) {
	now, err := normalizePaymentIntentOccurredAt(now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	intent, err := paymententity.NewProviderWithdrawalIntent(
		transactionID,
		provider,
		amount,
		feeAmount,
		currency,
		destinationAccountID,
		debitAccountID,
		now,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := newPaymentIntentAggregate(intent)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.recordPaymentCreated(metadata, now); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.recordWithdrawalRequested(now); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *PaymentIntentAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&PaymentCreatedEvent{},
		&PaymentWithdrawalRequestedEvent{},
		&PaymentCheckoutSessionCreatedEvent{},
		&PaymentProviderStateChangedEvent{},
		&PaymentSucceededEvent{},
		&PaymentFailedEvent{},
		&PaymentRefundedEvent{},
		&PaymentChargebackEvent{},
	)
}

func (r *PaymentIntentAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *PaymentCreatedEvent:
		return r.applyPaymentCreated(data)
	case PaymentCreatedEvent:
		return r.applyPaymentCreated(&data)
	case *PaymentWithdrawalRequestedEvent:
		return r.applyWithdrawalRequested(data)
	case PaymentWithdrawalRequestedEvent:
		return r.applyWithdrawalRequested(&data)
	case *PaymentCheckoutSessionCreatedEvent:
		return r.applyCheckoutSessionCreated(data)
	case PaymentCheckoutSessionCreatedEvent:
		return r.applyCheckoutSessionCreated(&data)
	case *PaymentProviderStateChangedEvent:
		return r.applyProviderStateChanged(data)
	case PaymentProviderStateChangedEvent:
		return r.applyProviderStateChanged(&data)
	case *PaymentSucceededEvent:
		return r.applyPaymentSucceeded(data)
	case PaymentSucceededEvent:
		return r.applyPaymentSucceeded(&data)
	case *PaymentFailedEvent:
		return r.applyPaymentFailed(data)
	case PaymentFailedEvent:
		return r.applyPaymentFailed(&data)
	case *PaymentRefundedEvent:
		return r.applyPaymentRefunded(data)
	case PaymentRefundedEvent:
		return r.applyPaymentRefunded(&data)
	case *PaymentChargebackEvent:
		return r.applyPaymentChargeback(data)
	case PaymentChargebackEvent:
		return r.applyPaymentChargeback(&data)
	default:
		return event.ErrUnsupportedEventType
	}
}

func RestorePaymentIntentAggregate(intent *paymententity.PaymentIntent) (*PaymentIntentAggregate, error) {
	return RestorePaymentIntentAggregateWithVersion(intent, 0)
}

func RestorePaymentIntentAggregateWithVersion(intent *paymententity.PaymentIntent, version int) (*PaymentIntentAggregate, error) {
	if intent == nil {
		return nil, stackErr.Error(paymententity.ErrPaymentTransactionIDRequired)
	}
	clone := *intent
	normalizePaymentIntentSnapshot(&clone)
	if version < 0 {
		version = 0
	}
	agg := &PaymentIntentAggregate{intent: &clone}
	agg.SetAggregateType(AggregateTypePaymentIntent)
	agg.SetInternal(clone.TransactionID, version, version)
	return agg, nil
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

func (a *PaymentIntentAggregate) Workflow() string {
	if a == nil || a.intent == nil {
		return ""
	}
	return a.intent.Workflow
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

func (a *PaymentIntentAggregate) PendingOutboxEvents() []event.Event {
	if a == nil {
		return nil
	}
	return a.CloneEvents()
}

func (a *PaymentIntentAggregate) MarkPersisted() {
	if a == nil {
		return
	}
	a.processedEvents = nil
	a.AggregateRoot.MarkPersisted()
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

	transition, err := a.transitionProviderState("", paymententity.PaymentStatusFailed, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || !transition.StateChanged {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	failedResult := a.currentProviderResult(paymententity.PaymentProviderResult{Status: paymententity.PaymentStatusFailed})
	if err := a.recordPaymentFailed(failedResult, a.intent.UpdatedAt); err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) applyNonFinalOutcome(
	result paymententity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
	occurredAt time.Time,
) (PaymentIntentMutation, error) {
	transition, err := a.transitionProviderResult(result, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || (!transition.StateChanged && !transition.ExternalRefChanged) {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	if err := a.recordProviderStateChanged(transition, result, occurredAt); err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if err := a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt); err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Type == paymententity.PaymentTransitionFailed {
		if err := a.recordPaymentFailed(a.currentProviderResult(result), occurredAt); err != nil {
			return PaymentIntentMutation{}, stackErr.Error(err)
		}
	}

	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) applySuccessfulOutcome(
	result paymententity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
	occurredAt time.Time,
) (PaymentIntentMutation, error) {
	transition, err := a.transitionProviderResult(result, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || transition.Type == paymententity.PaymentTransitionNone {
		if transition.ExternalRefChanged {
			if err := a.recordProviderStateChanged(transition, result, occurredAt); err != nil {
				return PaymentIntentMutation{}, stackErr.Error(err)
			}
			if err := a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt); err != nil {
				return PaymentIntentMutation{}, stackErr.Error(err)
			}
			return PaymentIntentMutation{Duplicate: true, Persist: true}, nil
		}
		return PaymentIntentMutation{Duplicate: true}, nil
	}
	if transition.Type != paymententity.PaymentTransitionSucceeded {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	processedEvent, err := a.newProcessedTransitionEvent(sharedevents.EventPaymentSucceeded, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}

	a.recordProcessedEvent(processedEvent)
	if err := a.recordPaymentSucceeded(a.currentProviderResult(result), occurredAt); err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if err := a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt); err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) applyReversedOutcome(
	result paymententity.PaymentProviderResult,
	checkoutURL string,
	emitCheckoutEvent bool,
	occurredAt time.Time,
) (PaymentIntentMutation, error) {
	transition, err := a.transitionProviderResult(result, occurredAt)
	if err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	if transition.Ignored || transition.Type == paymententity.PaymentTransitionNone {
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	var processedEvent *paymententity.ProcessedPaymentEvent
	switch transition.Type {
	case paymententity.PaymentTransitionRefunded:
		processedEvent, err = a.newProcessedTransitionEvent(sharedevents.EventPaymentRefunded, occurredAt)
		if err != nil {
			return PaymentIntentMutation{}, stackErr.Error(err)
		}
		a.recordProcessedEvent(processedEvent)
		if err := a.recordPaymentRefunded(a.currentProviderResult(result), occurredAt); err != nil {
			return PaymentIntentMutation{}, stackErr.Error(err)
		}
	case paymententity.PaymentTransitionChargeback:
		processedEvent, err = a.newProcessedTransitionEvent(sharedevents.EventPaymentChargeback, occurredAt)
		if err != nil {
			return PaymentIntentMutation{}, stackErr.Error(err)
		}
		a.recordProcessedEvent(processedEvent)
		if err := a.recordPaymentChargeback(a.currentProviderResult(result), occurredAt); err != nil {
			return PaymentIntentMutation{}, stackErr.Error(err)
		}
	default:
		return PaymentIntentMutation{Duplicate: true}, nil
	}

	if err := a.recordCheckoutSessionEvent(checkoutURL, emitCheckoutEvent, occurredAt); err != nil {
		return PaymentIntentMutation{}, stackErr.Error(err)
	}
	return PaymentIntentMutation{Persist: true}, nil
}

func (a *PaymentIntentAggregate) recordProcessedEvent(evt *paymententity.ProcessedPaymentEvent) {
	if a == nil || evt == nil {
		return
	}
	clone := *evt
	a.processedEvents = append(a.processedEvents, &clone)
}

func (a *PaymentIntentAggregate) recordPaymentCreated(metadata map[string]string, occurredAt time.Time) error {
	return a.applyChangeAt(&PaymentCreatedEvent{
		Workflow:           a.intent.Workflow,
		PaymentID:          a.intent.TransactionID,
		TransactionID:      a.intent.TransactionID,
		Provider:           a.intent.Provider,
		ClearingAccountKey: a.intent.ClearingAccountKey,
		Amount:             a.intent.Amount,
		FeeAmount:          a.intent.FeeAmount,
		ProviderAmount:     a.intent.ProviderAmount,
		Currency:           a.intent.Currency,
		DebitAccountID:     a.intent.DebitAccountID,
		CreditAccountID:    a.intent.CreditAccountID,
		Status:             a.intent.Status,
		Metadata:           metadata,
		CreatedAt:          occurredAt,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) recordWithdrawalRequested(occurredAt time.Time) error {
	return a.applyChangeAt(&PaymentWithdrawalRequestedEvent{
		PaymentID:            a.intent.TransactionID,
		TransactionID:        a.intent.TransactionID,
		Provider:             a.intent.Provider,
		ClearingAccountKey:   a.intent.ClearingAccountKey,
		DebitAccountID:       a.intent.DebitAccountID,
		DestinationAccountID: a.intent.DestinationAccountID,
		Amount:               a.intent.Amount,
		FeeAmount:            a.intent.FeeAmount,
		ProviderAmount:       a.intent.ProviderAmount,
		Currency:             a.intent.Currency,
		Status:               a.intent.Status,
		RequestedAt:          occurredAt,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) recordProviderStateChanged(transition paymententity.PaymentTransition, result paymententity.PaymentProviderResult, occurredAt time.Time) error {
	current := a.currentProviderResult(result)
	return a.applyChangeAt(&PaymentProviderStateChangedEvent{
		TransactionID:      a.intent.TransactionID,
		Provider:           a.intent.Provider,
		ProviderPaymentRef: current.ExternalRef,
		PreviousStatus:     transition.PreviousStatus,
		Status:             transition.CurrentStatus,
		ProviderEventID:    strings.TrimSpace(current.EventID),
		ProviderEventType:  strings.TrimSpace(current.EventType),
		Amount:             current.Amount,
		Currency:           current.Currency,
		OccurredAt:         occurredAt,
		StateChanged:       transition.StateChanged,
		ExternalRefChanged: transition.ExternalRefChanged,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) recordCheckoutSessionEvent(checkoutURL string, emitCheckoutEvent bool, occurredAt time.Time) error {
	if a == nil || a.intent == nil {
		return nil
	}
	if !emitCheckoutEvent || !a.shouldEmitCheckoutSessionCreated(checkoutURL) {
		return nil
	}
	return a.applyChangeAt(&PaymentCheckoutSessionCreatedEvent{
		Workflow:           a.intent.Workflow,
		PaymentID:          a.intent.TransactionID,
		TransactionID:      a.intent.TransactionID,
		Provider:           a.intent.Provider,
		ProviderPaymentRef: a.intent.ExternalRef,
		CheckoutURL:        strings.TrimSpace(checkoutURL),
		Amount:             a.intent.Amount,
		FeeAmount:          a.intent.FeeAmount,
		ProviderAmount:     a.intent.ProviderAmount,
		Currency:           a.intent.Currency,
		Status:             a.intent.Status,
		OccurredAt:         occurredAt,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) recordPaymentSucceeded(result paymententity.PaymentProviderResult, occurredAt time.Time) error {
	return a.applyChangeAt(&PaymentSucceededEvent{
		Workflow:           a.intent.Workflow,
		PaymentID:          a.intent.TransactionID,
		TransactionID:      a.intent.TransactionID,
		Provider:           a.intent.Provider,
		ClearingAccountKey: a.intent.ClearingAccountKey,
		DebitAccountID:     a.intent.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(result.EventID),
		ProviderEventType:  strings.TrimSpace(result.EventType),
		ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, a.intent.ExternalRef),
		Amount:             a.intent.Amount,
		FeeAmount:          a.intent.FeeAmount,
		ProviderAmount:     a.intent.ProviderAmount,
		Currency:           a.intent.Currency,
		CreditAccountID:    a.intent.CreditAccountID,
		IdempotencyKey:     a.transitionIdempotencyKey(sharedevents.EventPaymentSucceeded),
		SucceededAt:        occurredAt,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) recordPaymentFailed(result paymententity.PaymentProviderResult, occurredAt time.Time) error {
	return a.applyChangeAt(&PaymentFailedEvent{
		Workflow:           a.intent.Workflow,
		PaymentID:          a.intent.TransactionID,
		TransactionID:      a.intent.TransactionID,
		Provider:           a.intent.Provider,
		ClearingAccountKey: a.intent.ClearingAccountKey,
		DebitAccountID:     a.intent.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(result.EventID),
		ProviderEventType:  strings.TrimSpace(result.EventType),
		ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, a.intent.ExternalRef),
		Amount:             a.intent.Amount,
		FeeAmount:          a.intent.FeeAmount,
		ProviderAmount:     a.intent.ProviderAmount,
		Currency:           a.intent.Currency,
		CreditAccountID:    a.intent.CreditAccountID,
		Status:             paymententity.NormalizePaymentStatusOrPending(result.Status),
		OccurredAt:         occurredAt,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) recordPaymentRefunded(result paymententity.PaymentProviderResult, occurredAt time.Time) error {
	return a.applyChangeAt(&PaymentRefundedEvent{
		Workflow:           a.intent.Workflow,
		PaymentID:          a.intent.TransactionID,
		TransactionID:      a.intent.TransactionID,
		Provider:           a.intent.Provider,
		ClearingAccountKey: a.intent.ClearingAccountKey,
		DebitAccountID:     a.intent.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(result.EventID),
		ProviderEventType:  strings.TrimSpace(result.EventType),
		ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, a.intent.ExternalRef),
		Amount:             paymentResultAmountOrDefault(result.Amount, a.intent.Amount),
		FeeAmount:          a.intent.FeeAmount,
		ProviderAmount:     a.intent.ProviderAmount,
		Currency:           a.intent.Currency,
		CreditAccountID:    a.intent.CreditAccountID,
		IdempotencyKey:     a.transitionIdempotencyKey(sharedevents.EventPaymentRefunded),
		RefundedAt:         occurredAt,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) recordPaymentChargeback(result paymententity.PaymentProviderResult, occurredAt time.Time) error {
	return a.applyChangeAt(&PaymentChargebackEvent{
		Workflow:           a.intent.Workflow,
		PaymentID:          a.intent.TransactionID,
		TransactionID:      a.intent.TransactionID,
		Provider:           a.intent.Provider,
		ClearingAccountKey: a.intent.ClearingAccountKey,
		DebitAccountID:     a.intent.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(result.EventID),
		ProviderEventType:  strings.TrimSpace(result.EventType),
		ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, a.intent.ExternalRef),
		Amount:             paymentResultAmountOrDefault(result.Amount, a.intent.Amount),
		FeeAmount:          a.intent.FeeAmount,
		ProviderAmount:     a.intent.ProviderAmount,
		Currency:           a.intent.Currency,
		CreditAccountID:    a.intent.CreditAccountID,
		IdempotencyKey:     a.transitionIdempotencyKey(sharedevents.EventPaymentChargeback),
		ChargedBackAt:      occurredAt,
	}, occurredAt)
}

func (a *PaymentIntentAggregate) applyChangeAt(data interface{}, occurredAt time.Time) error {
	if err := a.ApplyChange(a, data); err != nil {
		return stackErr.Error(err)
	}
	events := a.Events()
	if len(events) == 0 {
		return nil
	}
	events[len(events)-1].CreatedAt = occurredAt.Unix()
	return nil
}

func (a *PaymentIntentAggregate) applyPaymentCreated(data *PaymentCreatedEvent) error {
	if data == nil {
		return nil
	}
	if a.intent == nil {
		a.intent = &paymententity.PaymentIntent{}
	}
	a.intent.Workflow = paymententity.NormalizePaymentWorkflow(data.Workflow)
	if a.intent.Workflow == "" {
		a.intent.Workflow = paymententity.PaymentWorkflowTopUp
	}
	a.intent.TransactionID = strings.TrimSpace(data.TransactionID)
	if a.intent.TransactionID == "" {
		a.intent.TransactionID = strings.TrimSpace(data.PaymentID)
	}
	a.intent.Provider = strings.ToLower(strings.TrimSpace(data.Provider))
	a.intent.ClearingAccountKey = effectivePaymentClearingAccountKey(a.intent.Provider, data.ClearingAccountKey)
	a.intent.Amount = data.Amount
	a.intent.FeeAmount = data.FeeAmount
	a.intent.ProviderAmount = data.ProviderAmount
	a.intent.Currency = strings.ToUpper(strings.TrimSpace(data.Currency))
	a.intent.DebitAccountID = strings.TrimSpace(data.DebitAccountID)
	a.intent.CreditAccountID = strings.TrimSpace(data.CreditAccountID)
	a.intent.Status = paymententity.NormalizePaymentStatusOrPending(data.Status)
	a.intent.CreatedAt = normalizePaymentTime(data.CreatedAt)
	a.intent.UpdatedAt = a.intent.CreatedAt
	return nil
}

func (a *PaymentIntentAggregate) applyWithdrawalRequested(data *PaymentWithdrawalRequestedEvent) error {
	if data == nil || a.intent == nil {
		return nil
	}
	a.intent.DestinationAccountID = strings.TrimSpace(data.DestinationAccountID)
	a.intent.DebitAccountID = strings.TrimSpace(data.DebitAccountID)
	a.intent.UpdatedAt = normalizePaymentTime(data.RequestedAt)
	return nil
}

func (a *PaymentIntentAggregate) applyCheckoutSessionCreated(data *PaymentCheckoutSessionCreatedEvent) error {
	if data == nil || a.intent == nil {
		return nil
	}
	if ref := strings.TrimSpace(data.ProviderPaymentRef); ref != "" {
		a.intent.ExternalRef = ref
	}
	a.intent.UpdatedAt = normalizePaymentTime(data.OccurredAt)
	return nil
}

func (a *PaymentIntentAggregate) applyProviderStateChanged(data *PaymentProviderStateChangedEvent) error {
	if data == nil || a.intent == nil {
		return nil
	}
	if ref := strings.TrimSpace(data.ProviderPaymentRef); ref != "" {
		a.intent.ExternalRef = ref
	}
	if status := paymententity.NormalizePaymentStatus(data.Status); status != "" {
		a.intent.Status = status
	}
	a.intent.UpdatedAt = normalizePaymentTime(data.OccurredAt)
	return nil
}

func (a *PaymentIntentAggregate) applyPaymentSucceeded(data *PaymentSucceededEvent) error {
	if data == nil || a.intent == nil {
		return nil
	}
	a.intent.Status = paymententity.PaymentStatusSuccess
	if ref := strings.TrimSpace(data.ProviderPaymentRef); ref != "" {
		a.intent.ExternalRef = ref
	}
	a.intent.UpdatedAt = normalizePaymentTime(data.SucceededAt)
	return nil
}

func (a *PaymentIntentAggregate) applyPaymentFailed(data *PaymentFailedEvent) error {
	if data == nil || a.intent == nil {
		return nil
	}
	a.intent.Status = paymententity.PaymentStatusFailed
	if ref := strings.TrimSpace(data.ProviderPaymentRef); ref != "" {
		a.intent.ExternalRef = ref
	}
	a.intent.UpdatedAt = normalizePaymentTime(data.OccurredAt)
	return nil
}

func (a *PaymentIntentAggregate) applyPaymentRefunded(data *PaymentRefundedEvent) error {
	if data == nil || a.intent == nil {
		return nil
	}
	a.intent.Status = paymententity.PaymentStatusRefunded
	if ref := strings.TrimSpace(data.ProviderPaymentRef); ref != "" {
		a.intent.ExternalRef = ref
	}
	a.intent.UpdatedAt = normalizePaymentTime(data.RefundedAt)
	return nil
}

func (a *PaymentIntentAggregate) applyPaymentChargeback(data *PaymentChargebackEvent) error {
	if data == nil || a.intent == nil {
		return nil
	}
	a.intent.Status = paymententity.PaymentStatusChargeback
	if ref := strings.TrimSpace(data.ProviderPaymentRef); ref != "" {
		a.intent.ExternalRef = ref
	}
	a.intent.UpdatedAt = normalizePaymentTime(data.ChargedBackAt)
	return nil
}

func (a *PaymentIntentAggregate) transitionProviderResult(result paymententity.PaymentProviderResult, updatedAt time.Time) (paymententity.PaymentTransition, error) {
	if a != nil && a.intent != nil {
		normalizePaymentIntentSnapshot(a.intent)
	}
	nextStatus := paymententity.NormalizePaymentStatusOrPending(result.Status)
	if err := a.validateProviderResultForStatus(nextStatus, result.Amount, result.Currency); err != nil {
		return paymententity.PaymentTransition{}, stackErr.Error(err)
	}

	return a.transitionProviderState(result.ExternalRef, nextStatus, updatedAt)
}

func (a *PaymentIntentAggregate) transitionProviderState(externalRef, status string, updatedAt time.Time) (paymententity.PaymentTransition, error) {
	if a == nil || a.intent == nil {
		return paymententity.PaymentTransition{}, paymententity.ErrPaymentTransactionIDRequired
	}
	normalizePaymentIntentSnapshot(a.intent)

	normalizedStatus := paymententity.NormalizePaymentStatus(status)
	if normalizedStatus == "" {
		return paymententity.PaymentTransition{}, paymententity.ErrPaymentStatusInvalid
	}

	transition := resolvePaymentTransition(paymententity.NormalizePaymentStatusOrPending(a.intent.Status), normalizedStatus)
	if transition.StateChanged {
		a.intent.Status = transition.CurrentStatus
	}

	if externalRef = strings.TrimSpace(externalRef); externalRef != "" && externalRef != a.intent.ExternalRef {
		a.intent.ExternalRef = externalRef
		transition.ExternalRefChanged = true
	}

	if transition.StateChanged || transition.ExternalRefChanged {
		a.intent.UpdatedAt = normalizePaymentTime(updatedAt)
	}

	return transition, nil
}

func resolvePaymentTransition(previousStatus, nextStatus string) paymententity.PaymentTransition {
	transition := paymententity.PaymentTransition{
		PreviousStatus: paymententity.NormalizePaymentStatusOrPending(previousStatus),
		CurrentStatus:  paymententity.NormalizePaymentStatusOrPending(previousStatus),
		Type:           paymententity.PaymentTransitionNone,
	}

	if transition.CurrentStatus == nextStatus {
		return transition
	}

	switch transition.CurrentStatus {
	case paymententity.PaymentStatusCreating:
		switch nextStatus {
		case paymententity.PaymentStatusPending:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionPending
			transition.StateChanged = true
		case paymententity.PaymentStatusSuccess:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionSucceeded
			transition.StateChanged = true
		case paymententity.PaymentStatusFailed:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionFailed
			transition.StateChanged = true
		case paymententity.PaymentStatusCancelled:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionCancelled
			transition.StateChanged = true
		default:
			transition.Ignored = true
		}
	case paymententity.PaymentStatusPending:
		switch nextStatus {
		case paymententity.PaymentStatusSuccess:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionSucceeded
			transition.StateChanged = true
		case paymententity.PaymentStatusFailed:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionFailed
			transition.StateChanged = true
		case paymententity.PaymentStatusCancelled:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionCancelled
			transition.StateChanged = true
		default:
			transition.Ignored = true
		}
	case paymententity.PaymentStatusSuccess:
		switch nextStatus {
		case paymententity.PaymentStatusRefunded:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionRefunded
			transition.StateChanged = true
		case paymententity.PaymentStatusChargeback:
			transition.CurrentStatus = nextStatus
			transition.Type = paymententity.PaymentTransitionChargeback
			transition.StateChanged = true
		default:
			transition.Ignored = true
		}
	case paymententity.PaymentStatusFailed, paymententity.PaymentStatusCancelled, paymententity.PaymentStatusRefunded, paymententity.PaymentStatusChargeback:
		transition.Ignored = true
	default:
		transition.Ignored = true
	}

	return transition
}

func (a *PaymentIntentAggregate) validateProviderResultForStatus(status string, amount int64, currency string) error {
	if a == nil || a.intent == nil {
		return paymententity.ErrPaymentTransactionIDRequired
	}

	normalizedStatus := paymententity.NormalizePaymentStatus(status)
	if normalizedStatus == "" {
		normalizedStatus = paymententity.NormalizePaymentStatusOrPending(status)
	}
	switch normalizedStatus {
	case paymententity.PaymentStatusRefunded, paymententity.PaymentStatusChargeback:
		if amount != 0 && amount > a.intent.ProviderAmount {
			return paymententity.ErrPaymentProviderAmountMismatch
		}
	default:
		if amount != 0 && amount != a.intent.ProviderAmount {
			return paymententity.ErrPaymentProviderAmountMismatch
		}
	}

	if currency = strings.TrimSpace(currency); currency != "" && !strings.EqualFold(currency, a.intent.Currency) {
		return paymententity.ErrPaymentProviderCurrencyMismatch
	}
	return nil
}

func (a *PaymentIntentAggregate) paymentIdempotencyKey(eventID, externalRef string) string {
	if eventID = strings.TrimSpace(eventID); eventID != "" {
		return eventID
	}
	if externalRef = strings.TrimSpace(externalRef); externalRef != "" {
		return externalRef
	}
	if a != nil && a.intent != nil {
		if externalRef = strings.TrimSpace(a.intent.ExternalRef); externalRef != "" {
			return externalRef
		}
		return strings.TrimSpace(a.intent.TransactionID)
	}
	return ""
}

func (a *PaymentIntentAggregate) transitionIdempotencyKey(eventName string) string {
	if a == nil || a.intent == nil {
		return strings.TrimSpace(eventName) + ":"
	}
	return fmt.Sprintf("%s:%s", strings.TrimSpace(eventName), strings.TrimSpace(a.intent.TransactionID))
}

func (a *PaymentIntentAggregate) newProcessedTransitionEvent(eventName string, createdAt time.Time) (*paymententity.ProcessedPaymentEvent, error) {
	return paymententity.NewProcessedPaymentEvent(
		a.intent.Provider,
		a.transitionIdempotencyKey(eventName),
		a.intent.TransactionID,
		createdAt,
	)
}

func (a *PaymentIntentAggregate) currentProviderResult(source paymententity.PaymentProviderResult) paymententity.PaymentProviderResult {
	if a == nil || a.intent == nil {
		return paymententity.PaymentProviderResult{}
	}

	amount := source.Amount
	if amount == 0 {
		amount = a.intent.ProviderAmount
	}

	return paymententity.PaymentProviderResult{
		TransactionID: coalescePaymentValue(source.TransactionID, a.intent.TransactionID),
		EventID:       strings.TrimSpace(source.EventID),
		EventType:     strings.TrimSpace(source.EventType),
		Status:        paymententity.NormalizePaymentStatusOrPending(coalescePaymentValue(source.Status, a.intent.Status)),
		Amount:        amount,
		Currency:      coalescePaymentValue(source.Currency, a.intent.Currency),
		ExternalRef:   coalescePaymentValue(source.ExternalRef, a.intent.ExternalRef),
	}
}

func (a *PaymentIntentAggregate) shouldEmitCheckoutSessionCreated(checkoutURL string) bool {
	if a == nil || a.intent == nil {
		return false
	}
	if a.intent.Workflow != paymententity.PaymentWorkflowTopUp {
		return false
	}
	return strings.TrimSpace(checkoutURL) != "" || strings.TrimSpace(a.intent.ExternalRef) != ""
}

func newPaymentIntentAggregate(intent *paymententity.PaymentIntent) (*PaymentIntentAggregate, error) {
	if intent == nil {
		return nil, paymententity.ErrPaymentTransactionIDRequired
	}
	agg := &PaymentIntentAggregate{intent: intent}
	if err := event.InitAggregate(&agg.AggregateRoot, agg, intent.TransactionID); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func normalizePaymentIntentSnapshot(intent *paymententity.PaymentIntent) {
	if intent == nil {
		return
	}
	intent.TransactionID = strings.TrimSpace(intent.TransactionID)
	if intent.Workflow = paymententity.NormalizePaymentWorkflow(intent.Workflow); intent.Workflow == "" {
		intent.Workflow = paymententity.PaymentWorkflowTopUp
	}
	intent.Provider = strings.ToLower(strings.TrimSpace(intent.Provider))
	intent.ExternalRef = strings.TrimSpace(intent.ExternalRef)
	intent.DestinationAccountID = strings.TrimSpace(intent.DestinationAccountID)
	intent.Currency = strings.ToUpper(strings.TrimSpace(intent.Currency))
	intent.ClearingAccountKey = effectivePaymentClearingAccountKey(intent.Provider, intent.ClearingAccountKey)
	intent.DebitAccountID = strings.TrimSpace(intent.DebitAccountID)
	intent.CreditAccountID = strings.TrimSpace(intent.CreditAccountID)
	if intent.ProviderAmount <= 0 {
		switch intent.Workflow {
		case paymententity.PaymentWorkflowWithdrawal:
			intent.ProviderAmount = intent.Amount
		default:
			intent.ProviderAmount = intent.Amount + intent.FeeAmount
		}
	}
	if intent.Status = paymententity.NormalizePaymentStatus(intent.Status); intent.Status == "" {
		intent.Status = paymententity.PaymentStatusCreating
	}
	intent.CreatedAt = normalizePaymentTime(intent.CreatedAt)
	intent.UpdatedAt = normalizePaymentTime(intent.UpdatedAt)
}

func providerClearingAccountKey(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return ""
	}
	return fmt.Sprintf("provider:%s", provider)
}

func effectivePaymentClearingAccountKey(provider, clearingAccountKey string) string {
	if clearingAccountKey = strings.TrimSpace(clearingAccountKey); clearingAccountKey != "" {
		return clearingAccountKey
	}
	return providerClearingAccountKey(provider)
}

func normalizePaymentTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func coalescePaymentValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func paymentResultAmountOrDefault(amount int64, fallback int64) int64 {
	if amount != 0 {
		return amount
	}
	return fallback
}

func normalizePaymentIntentOccurredAt(value time.Time) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, ErrPaymentIntentOccurredAtRequired
	}
	return value.UTC(), nil
}
