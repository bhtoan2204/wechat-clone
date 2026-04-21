package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ledgerservice "wechat-clone/core/modules/ledger/application/service"
	ledgerentity "wechat-clone/core/modules/ledger/domain/entity"
	"wechat-clone/core/shared/contracts"
	sharedevents "wechat-clone/core/shared/contracts/events"
	sharedlock "wechat-clone/core/shared/infra/lock"
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
		command := ledgerservice.RecordPaymentSucceededCommand{
			PaymentID:          payload.PaymentID,
			TransactionID:      payload.TransactionID,
			ClearingAccountKey: payload.ClearingAccountKey,
			CreditAccountID:    payload.CreditAccountID,
			Currency:           payload.Currency,
			Amount:             payload.Amount,
		}

		lockKeys, err := paymentSucceededAccountLockKeys(command)
		if err != nil {
			return stackErr.Error(h.ledgerService.RecordPaymentSucceeded(ctx, command))
		}

		return stackErr.Error(h.withLedgerAccountLocks(ctx, lockKeys, func() error {
			return h.ledgerService.RecordPaymentSucceeded(ctx, command)
		}))
	case sharedevents.EventPaymentRefunded:
		payload, err := unmarshalPaymentRefundedPayload(event.EventData)
		if err != nil {
			return stackErr.Error(err)
		}
		payload.PaymentID = resolvePaymentRefundedID(event.AggregateID, payload)
		command := ledgerservice.RecordPaymentReversedCommand{
			PaymentID:          payload.PaymentID,
			TransactionID:      payload.TransactionID,
			ClearingAccountKey: payload.ClearingAccountKey,
			CreditAccountID:    payload.CreditAccountID,
			Currency:           payload.Currency,
			Amount:             payload.Amount,
			ReversalType:       ledgerentity.PaymentReferenceRefunded,
		}

		lockKeys, err := paymentReversedAccountLockKeys(command)
		if err != nil {
			return stackErr.Error(h.ledgerService.RecordPaymentReversed(ctx, command))
		}

		return stackErr.Error(h.withLedgerAccountLocks(ctx, lockKeys, func() error {
			return h.ledgerService.RecordPaymentReversed(ctx, command)
		}))
	case sharedevents.EventPaymentChargeback:
		payload, err := unmarshalPaymentChargebackPayload(event.EventData)
		if err != nil {
			return stackErr.Error(err)
		}
		payload.PaymentID = resolvePaymentChargebackID(event.AggregateID, payload)
		command := ledgerservice.RecordPaymentReversedCommand{
			PaymentID:          payload.PaymentID,
			TransactionID:      payload.TransactionID,
			ClearingAccountKey: payload.ClearingAccountKey,
			CreditAccountID:    payload.CreditAccountID,
			Currency:           payload.Currency,
			Amount:             payload.Amount,
			ReversalType:       ledgerentity.PaymentReferenceChargeback,
		}

		lockKeys, err := paymentReversedAccountLockKeys(command)
		if err != nil {
			return stackErr.Error(h.ledgerService.RecordPaymentReversed(ctx, command))
		}

		return stackErr.Error(h.withLedgerAccountLocks(ctx, lockKeys, func() error {
			return h.ledgerService.RecordPaymentReversed(ctx, command)
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

func paymentSucceededAccountLockKeys(command ledgerservice.RecordPaymentSucceededCommand) ([]string, error) {
	booking, err := ledgerentity.NewPaymentSucceededBooking(ledgerentity.PaymentSucceededBookingInput{
		PaymentID:          command.PaymentID,
		TransactionID:      command.TransactionID,
		ClearingAccountKey: command.ClearingAccountKey,
		CreditAccountID:    command.CreditAccountID,
		Currency:           command.Currency,
		Amount:             command.Amount,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return []string{booking.DebitAccountID, booking.CreditAccountID}, nil
}

func paymentReversedAccountLockKeys(command ledgerservice.RecordPaymentReversedCommand) ([]string, error) {
	booking, err := ledgerentity.NewPaymentReversalBooking(ledgerentity.PaymentReversalBookingInput{
		PaymentID:          command.PaymentID,
		TransactionID:      command.TransactionID,
		ClearingAccountKey: command.ClearingAccountKey,
		CreditAccountID:    command.CreditAccountID,
		Currency:           command.Currency,
		Amount:             command.Amount,
		ReversalType:       command.ReversalType,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return []string{booking.DebitAccountID, booking.CreditAccountID}, nil
}
