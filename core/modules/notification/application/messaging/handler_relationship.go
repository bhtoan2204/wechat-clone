package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"wechat-clone/core/modules/notification/application/support"
	"wechat-clone/core/modules/notification/domain/aggregate"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	notificationtypes "wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/contracts"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (h *messageHandler) handleRelationshipOutboxEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("handleRelationshipOutboxEvent")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal relationship outbox event failed: %w", err))
	}

	log.Infow("handle relationship outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case sharedevents.EventRelationshipPairFriendRequestSent:
		return stackErr.Error(h.handleFriendRequestSentEvent(ctx, event.EventData))
	case sharedevents.EventRelationshipPairFriendRequestCancelled:
		return stackErr.Error(h.handleFriendRequestCancelledEvent(ctx, event.EventData))
	case sharedevents.EventRelationshipPairFriendRequestAccepted:
		return stackErr.Error(h.handleFriendRequestAcceptedEvent(ctx, event.EventData))
	case sharedevents.EventRelationshipPairFriendRequestRejected:
		return stackErr.Error(h.handleFriendRequestRejectedEvent(ctx, event.EventData))
	default:
		return nil
	}
}

func (h *messageHandler) handleFriendRequestSentEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventRelationshipPairFriendRequestSent, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode friend request sent payload failed: %w", err))
	}
	payload, ok := payloadAny.(*sharedevents.RelationshipPairFriendRequestSentEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventRelationshipPairFriendRequestSent))
	}

	return stackErr.Error(h.createGeneralNotificationAndEmit(ctx, generalNotificationSpec{
		NotificationID: aggregate.FriendRequestNotificationID(notificationtypes.NotificationTypeFriendRequestSent, payload.RequestID, payload.AddresseeID),
		AccountID:      payload.AddresseeID,
		Type:           notificationtypes.NotificationTypeFriendRequestSent,
		Subject:        "New friend request",
		Body:           "You have a new friend request waiting for your response.",
		OccurredAt:     payload.CreatedAt,
	}))
}

func (h *messageHandler) handleFriendRequestCancelledEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventRelationshipPairFriendRequestCancelled, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode friend request cancelled payload failed: %w", err))
	}
	payload, ok := payloadAny.(*sharedevents.RelationshipPairFriendRequestCancelledEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventRelationshipPairFriendRequestCancelled))
	}

	return stackErr.Error(h.createGeneralNotificationAndEmit(ctx, generalNotificationSpec{
		NotificationID: aggregate.FriendRequestNotificationID(notificationtypes.NotificationTypeFriendRequestCancelled, payload.RequestID, payload.AddresseeID),
		AccountID:      payload.AddresseeID,
		Type:           notificationtypes.NotificationTypeFriendRequestCancelled,
		Subject:        "Friend request cancelled",
		Body:           "A friend request was cancelled before you responded.",
		OccurredAt:     payload.CancelledAt,
	}))
}

func (h *messageHandler) handleFriendRequestAcceptedEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventRelationshipPairFriendRequestAccepted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode friend request accepted payload failed: %w", err))
	}
	payload, ok := payloadAny.(*sharedevents.RelationshipPairFriendRequestAcceptedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventRelationshipPairFriendRequestAccepted))
	}

	return stackErr.Error(h.createGeneralNotificationAndEmit(ctx, generalNotificationSpec{
		NotificationID: aggregate.FriendRequestNotificationID(notificationtypes.NotificationTypeFriendRequestAccepted, payload.RequestID, payload.RequesterID),
		AccountID:      payload.RequesterID,
		Type:           notificationtypes.NotificationTypeFriendRequestAccepted,
		Subject:        "Friend request accepted",
		Body:           "Your friend request was accepted. You can now start chatting.",
		OccurredAt:     payload.AcceptedAt,
	}))
}

func (h *messageHandler) handleFriendRequestRejectedEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventRelationshipPairFriendRequestRejected, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode friend request rejected payload failed: %w", err))
	}
	payload, ok := payloadAny.(*sharedevents.RelationshipPairFriendRequestRejectedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventRelationshipPairFriendRequestRejected))
	}

	body := "Your friend request was declined."
	if payload.Reason != nil && strings.TrimSpace(*payload.Reason) != "" {
		body = "Your friend request was declined."
	}

	return stackErr.Error(h.createGeneralNotificationAndEmit(ctx, generalNotificationSpec{
		NotificationID: aggregate.FriendRequestNotificationID(notificationtypes.NotificationTypeFriendRequestRejected, payload.RequestID, payload.RequesterID),
		AccountID:      payload.RequesterID,
		Type:           notificationtypes.NotificationTypeFriendRequestRejected,
		Subject:        "Friend request declined",
		Body:           body,
		OccurredAt:     payload.RejectedAt,
	}))
}

type generalNotificationSpec struct {
	NotificationID string
	AccountID      string
	Type           notificationtypes.NotificationType
	Subject        string
	Body           string
	OccurredAt     time.Time
}

func (h *messageHandler) createGeneralNotificationAndEmit(ctx context.Context, spec generalNotificationSpec) error {
	if strings.TrimSpace(spec.NotificationID) == "" || strings.TrimSpace(spec.AccountID) == "" {
		return nil
	}

	notificationRepo := h.baseRepo.NotificationRepository()
	if _, err := notificationRepo.Load(ctx, spec.NotificationID); err == nil {
		return nil
	} else if !errors.Is(err, notificationrepos.ErrNotificationNotFound) {
		return stackErr.Error(err)
	}

	notificationAgg, err := aggregate.NewNotificationAggregate(spec.NotificationID)
	if err != nil {
		return stackErr.Error(err)
	}

	if err := notificationAgg.Create(
		spec.AccountID,
		spec.Type,
		spec.Subject,
		spec.Body,
		spec.OccurredAt,
	); err != nil {
		return stackErr.Error(err)
	}
	if err := notificationRepo.Save(ctx, notificationAgg); err != nil {
		return stackErr.Error(fmt.Errorf("save relationship notification failed: %w", err))
	}

	snapshot, err := notificationAgg.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}
	unreadCount, err := notificationRepo.CountUnread(ctx, spec.AccountID)
	if err != nil {
		return stackErr.Error(err)
	}
	if h.realtime != nil {
		if emitErr := h.realtime.EmitMessage(ctx, support.NewRealtimeNotificationPayload(notificationtypes.RealtimeEventNotificationUpsert, snapshot, unreadCount)); emitErr != nil {
			logging.FromContext(ctx).Warnw("emit relationship notification realtime failed", zap.Error(emitErr))
		}
	}
	if h.push != nil {
		if pushErr := h.push.SendNotification(ctx, snapshot); pushErr != nil {
			logging.FromContext(ctx).Warnw("send relationship notification webpush failed", zap.Error(pushErr))
		}
	}

	return nil
}
