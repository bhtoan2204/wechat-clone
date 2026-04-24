package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"wechat-clone/core/modules/notification/domain/aggregate"
	notificationtypes "wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/contracts"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/stackErr"
)

func (h *messageHandler) handlePaymentOutboxEvent(ctx context.Context, value []byte) error {
	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal payment outbox event failed: %w", err))
	}

	switch event.EventName {
	case sharedevents.EventPaymentWithdrawalRequested:
		return stackErr.Error(h.handlePaymentWithdrawalRequestedEvent(ctx, event.EventData))
	case sharedevents.EventPaymentSucceeded:
		return stackErr.Error(h.handlePaymentSucceededEvent(ctx, event.EventData))
	case sharedevents.EventPaymentFailed:
		return stackErr.Error(h.handlePaymentFailedEvent(ctx, event.EventData))
	default:
		return nil
	}
}

func (h *messageHandler) handlePaymentWithdrawalRequestedEvent(ctx context.Context, raw json.RawMessage) error {
	var payload sharedevents.PaymentWithdrawalRequestedEvent
	if err := contracts.UnmarshalEventData(raw, &payload); err != nil {
		return stackErr.Error(fmt.Errorf("decode payment withdrawal requested payload failed: %w", err))
	}

	return stackErr.Error(h.createGeneralNotificationAndEmit(ctx, generalNotificationSpec{
		NotificationID: aggregate.PaymentNotificationID(notificationtypes.NotificationTypeWithdrawalRequested, payload.PaymentID, payload.DebitAccountID),
		AccountID:      payload.DebitAccountID,
		Type:           notificationtypes.NotificationTypeWithdrawalRequested,
		Subject:        "Withdrawal requested",
		Body:           fmt.Sprintf("Your withdrawal request for %d %s has been received and is being processed.", payload.Amount, payload.Currency),
		OccurredAt:     payload.RequestedAt,
	}))
}

func (h *messageHandler) handlePaymentSucceededEvent(ctx context.Context, raw json.RawMessage) error {
	var payload sharedevents.PaymentSucceededEvent
	if err := contracts.UnmarshalEventData(raw, &payload); err != nil {
		return stackErr.Error(fmt.Errorf("decode payment succeeded payload failed: %w", err))
	}
	if payload.Workflow != "WITHDRAWAL" {
		return nil
	}

	return stackErr.Error(h.createGeneralNotificationAndEmit(ctx, generalNotificationSpec{
		NotificationID: aggregate.PaymentNotificationID(notificationtypes.NotificationTypeWithdrawalSucceeded, payload.PaymentID, payload.DebitAccountID),
		AccountID:      payload.DebitAccountID,
		Type:           notificationtypes.NotificationTypeWithdrawalSucceeded,
		Subject:        "Withdrawal completed",
		Body:           fmt.Sprintf("Your withdrawal of %d %s completed successfully.", payload.Amount, payload.Currency),
		OccurredAt:     payload.SucceededAt,
	}))
}

func (h *messageHandler) handlePaymentFailedEvent(ctx context.Context, raw json.RawMessage) error {
	var payload sharedevents.PaymentFailedEvent
	if err := contracts.UnmarshalEventData(raw, &payload); err != nil {
		return stackErr.Error(fmt.Errorf("decode payment failed payload failed: %w", err))
	}
	if payload.Workflow != "WITHDRAWAL" {
		return nil
	}

	return stackErr.Error(h.createGeneralNotificationAndEmit(ctx, generalNotificationSpec{
		NotificationID: aggregate.PaymentNotificationID(notificationtypes.NotificationTypeWithdrawalFailed, payload.PaymentID, payload.DebitAccountID),
		AccountID:      payload.DebitAccountID,
		Type:           notificationtypes.NotificationTypeWithdrawalFailed,
		Subject:        "Withdrawal failed",
		Body:           fmt.Sprintf("Your withdrawal of %d %s failed and the reserved balance was returned.", payload.Amount, payload.Currency),
		OccurredAt:     payload.OccurredAt,
	}))
}
