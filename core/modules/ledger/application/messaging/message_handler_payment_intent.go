package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ledgerservice "wechat-clone/core/modules/ledger/application/service"
	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	ledgerentity "wechat-clone/core/modules/ledger/domain/entity"
	valueobject "wechat-clone/core/modules/ledger/domain/value_object"
	"wechat-clone/core/shared/contracts"
	sharedevents "wechat-clone/core/shared/contracts/events"
	sharedlock "wechat-clone/core/shared/infra/lock"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (h *messageHandler) handlePaymentOutboxEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("LedgerPaymentEvent")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal payment outbox event failed: %w", err))
	}

	log.Infow("handle payment outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case sharedevents.EventPaymentSucceeded:
		payload, err := unmarshalPaymentSucceededPayload(event.EventData)
		if err != nil {
			return stackErr.Error(err)
		}
		payload.PaymentID = resolvePaymentSucceededID(event.AggregateID, payload)
		events, err := paymentSucceededLedgerEvents(payload)
		if err != nil {
			return stackErr.Error(err)
		}

		return stackErr.Error(h.withLedgerAccountLocks(ctx, ledgerEventLockKeys(events), func() error {
			return h.ledgerService.RecordLedgerEvents(ctx, ledgerservice.RecordLedgerEventsCommand{Events: events})
		}))
	case sharedevents.EventPaymentRefunded:
		payload, err := unmarshalPaymentRefundedPayload(event.EventData)
		if err != nil {
			return stackErr.Error(err)
		}
		payload.PaymentID = resolvePaymentRefundedID(event.AggregateID, payload)
		events, err := paymentReversedLedgerEvents(payload.PaymentID, payload.TransactionID, payload.ClearingAccountKey, payload.CreditAccountID, payload.Currency, payload.Amount, sharedevents.EventPaymentRefunded, payload.RefundedAt)
		if err != nil {
			return stackErr.Error(err)
		}

		return stackErr.Error(h.withLedgerAccountLocks(ctx, ledgerEventLockKeys(events), func() error {
			return h.ledgerService.RecordLedgerEvents(ctx, ledgerservice.RecordLedgerEventsCommand{Events: events})
		}))
	case sharedevents.EventPaymentChargeback:
		payload, err := unmarshalPaymentChargebackPayload(event.EventData)
		if err != nil {
			return stackErr.Error(err)
		}
		payload.PaymentID = resolvePaymentChargebackID(event.AggregateID, payload)
		events, err := paymentReversedLedgerEvents(payload.PaymentID, payload.TransactionID, payload.ClearingAccountKey, payload.CreditAccountID, payload.Currency, payload.Amount, sharedevents.EventPaymentChargeback, payload.ChargedBackAt)
		if err != nil {
			return stackErr.Error(err)
		}

		return stackErr.Error(h.withLedgerAccountLocks(ctx, ledgerEventLockKeys(events), func() error {
			return h.ledgerService.RecordLedgerEvents(ctx, ledgerservice.RecordLedgerEventsCommand{Events: events})
		}))
	default:
		return nil
	}
}

func (h *messageHandler) withLedgerAccountLocks(ctx context.Context, lockKeys []string, fn func() error) error {
	opts := sharedlock.DefaultMultiLockOptions()
	opts.KeyPrefix = ledgerservice.LedgerAccountLockKeyPrefix

	_, err := sharedlock.WithLocks(ctx, h.locker, lockKeys, opts, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	if err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func unmarshalPaymentSucceededPayload(data json.RawMessage) (sharedevents.PaymentSucceededEvent, error) {
	var payload sharedevents.PaymentSucceededEvent
	if err := contracts.UnmarshalEventData(data, &payload); err != nil {
		return sharedevents.PaymentSucceededEvent{}, stackErr.Error(fmt.Errorf("unmarshal payment succeeded payload failed: %w", err))
	}
	return payload, nil
}

func unmarshalPaymentRefundedPayload(data json.RawMessage) (sharedevents.PaymentRefundedEvent, error) {
	var payload sharedevents.PaymentRefundedEvent
	if err := contracts.UnmarshalEventData(data, &payload); err != nil {
		return sharedevents.PaymentRefundedEvent{}, stackErr.Error(fmt.Errorf("unmarshal payment refunded payload failed: %w", err))
	}
	return payload, nil
}

func unmarshalPaymentChargebackPayload(data json.RawMessage) (sharedevents.PaymentChargebackEvent, error) {
	var payload sharedevents.PaymentChargebackEvent
	if err := contracts.UnmarshalEventData(data, &payload); err != nil {
		return sharedevents.PaymentChargebackEvent{}, stackErr.Error(fmt.Errorf("unmarshal payment chargeback payload failed: %w", err))
	}
	return payload, nil
}

func resolvePaymentSucceededID(aggregateID string, payload sharedevents.PaymentSucceededEvent) string {
	paymentID := strings.TrimSpace(payload.PaymentID)
	if paymentID != "" {
		return paymentID
	}

	paymentID = strings.TrimSpace(aggregateID)
	if paymentID != "" {
		return paymentID
	}

	return strings.TrimSpace(payload.TransactionID)
}

func resolvePaymentRefundedID(aggregateID string, payload sharedevents.PaymentRefundedEvent) string {
	paymentID := strings.TrimSpace(payload.PaymentID)
	if paymentID != "" {
		return paymentID
	}
	paymentID = strings.TrimSpace(aggregateID)
	if paymentID != "" {
		return paymentID
	}
	return strings.TrimSpace(payload.TransactionID)
}

func resolvePaymentChargebackID(aggregateID string, payload sharedevents.PaymentChargebackEvent) string {
	paymentID := strings.TrimSpace(payload.PaymentID)
	if paymentID != "" {
		return paymentID
	}
	paymentID = strings.TrimSpace(aggregateID)
	if paymentID != "" {
		return paymentID
	}
	return strings.TrimSpace(payload.TransactionID)
}

func paymentSucceededLedgerEvents(payload sharedevents.PaymentSucceededEvent) ([]eventpkg.Event, error) {
	booking, err := ledgerentity.NewPaymentSucceededBooking(ledgerentity.PaymentSucceededBookingInput{
		PaymentID:          payload.PaymentID,
		TransactionID:      payload.TransactionID,
		ClearingAccountKey: payload.ClearingAccountKey,
		CreditAccountID:    payload.CreditAccountID,
		Currency:           payload.Currency,
		Amount:             payload.Amount,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return paymentLedgerEventsFromBooking(
		booking.LedgerTransactionID(),
		booking.PaymentID,
		booking.Currency,
		booking.Amount,
		booking.DebitAccountID,
		booking.CreditAccountID,
		ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent,
		ledgeraggregate.EventNameLedgerAccountDepositFromIntent,
		payload.SucceededAt,
	)
}

func paymentReversedLedgerEvents(paymentID, transactionID, clearingAccountKey, creditAccountID, currency string, amount int64, reversalType string, bookedAt time.Time) ([]eventpkg.Event, error) {
	booking, err := ledgerentity.NewPaymentReversalBooking(ledgerentity.PaymentReversalBookingInput{
		PaymentID:          paymentID,
		TransactionID:      transactionID,
		ClearingAccountKey: clearingAccountKey,
		CreditAccountID:    creditAccountID,
		Currency:           currency,
		Amount:             amount,
		ReversalType:       reversalType,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return paymentLedgerEventsFromBooking(
		booking.LedgerTransactionID(),
		booking.PaymentID,
		booking.Currency,
		booking.Amount,
		booking.DebitAccountID,
		booking.CreditAccountID,
		debitLedgerEventNameForReversal(booking.ReversalType),
		creditLedgerEventNameForReversal(booking.ReversalType),
		bookedAt,
	)
}

func paymentLedgerEventsFromBooking(transactionID, paymentID, currency string, amount int64, debitAccountID, creditAccountID, debitEventName, creditEventName string, bookedAt time.Time) ([]eventpkg.Event, error) {
	if bookedAt.IsZero() {
		bookedAt = time.Now().UTC()
	}
	debitPosting, err := ledgeraggregate.NewLedgerAccountPaymentPosting(
		valueobject.LedgerAccountPostingInput{
			AccountID:             debitAccountID,
			TransactionID:         transactionID,
			ReferenceType:         debitEventName,
			ReferenceID:           paymentID,
			CounterpartyAccountID: creditAccountID,
			Currency:              currency,
			AmountDelta:           -amount,
			BookedAt:              bookedAt,
		},
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	creditPosting, err := ledgeraggregate.NewLedgerAccountPaymentPosting(
		valueobject.LedgerAccountPostingInput{
			AccountID:             creditAccountID,
			TransactionID:         transactionID,
			ReferenceType:         creditEventName,
			ReferenceID:           paymentID,
			CounterpartyAccountID: debitAccountID,
			Currency:              currency,
			AmountDelta:           amount,
			BookedAt:              bookedAt,
		},
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	debitEvent, ok, err := ledgeraggregate.NewLedgerAccountEventFromPosting(debitAccountID, debitPosting)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if !ok {
		return nil, stackErr.Error(fmt.Errorf("unsupported debit ledger event reference_type=%s", debitEventName))
	}
	creditEvent, ok, err := ledgeraggregate.NewLedgerAccountEventFromPosting(creditAccountID, creditPosting)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if !ok {
		return nil, stackErr.Error(fmt.Errorf("unsupported credit ledger event reference_type=%s", creditEventName))
	}

	return []eventpkg.Event{debitEvent, creditEvent}, nil
}

func debitLedgerEventNameForReversal(paymentEventName string) string {
	switch strings.TrimSpace(paymentEventName) {
	case sharedevents.EventPaymentRefunded:
		return ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund
	case sharedevents.EventPaymentChargeback:
		return ledgeraggregate.EventNameLedgerAccountWithdrawFromChargeback
	default:
		return ""
	}
}

func creditLedgerEventNameForReversal(paymentEventName string) string {
	switch strings.TrimSpace(paymentEventName) {
	case sharedevents.EventPaymentRefunded:
		return ledgeraggregate.EventNameLedgerAccountDepositFromRefund
	case sharedevents.EventPaymentChargeback:
		return ledgeraggregate.EventNameLedgerAccountDepositFromChargeback
	default:
		return ""
	}
}

func ledgerEventLockKeys(events []eventpkg.Event) []string {
	keys := make([]string, 0, len(events))
	for _, evt := range events {
		keys = append(keys, strings.TrimSpace(evt.AggregateID))
	}
	return keys
}
