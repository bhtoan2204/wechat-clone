package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"wechat-clone/core/modules/notification/application/support"
	"wechat-clone/core/modules/notification/domain/aggregate"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	notificationtypes "wechat-clone/core/modules/notification/types"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (h *messageHandler) handleAccountCreatedEvent(ctx context.Context, raw json.RawMessage) error {
	log := logging.FromContext(ctx).Named("handleAccountCreatedEvent")
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventAccountCreated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode event payload failed: %w", err))
	}

	payload, ok := payloadAny.(*sharedevents.AccountCreatedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventAccountCreated))
	}

	subject := "Welcome to Go Socket"
	body := fmt.Sprintf("Welcome %s!", payload.Email)
	notificationID := aggregate.WelcomeNotificationID(payload.AccountID)

	notificationRepo := h.baseRepo.NotificationRepository()
	if _, err := notificationRepo.Load(ctx, notificationID); err == nil {
		return nil
	} else if !errors.Is(err, notificationrepos.ErrNotificationNotFound) {
		log.Errorw("load notification failed", zap.Error(err))
		return stackErr.Error(fmt.Errorf("load notification failed: %w", err))
	}

	notificationAgg, err := aggregate.NewNotificationAggregate(notificationID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := notificationAgg.Create(
		payload.AccountID,
		notificationtypes.NotificationTypeAccountCreated,
		subject,
		body,
		payload.CreatedAt,
	); err != nil {
		return stackErr.Error(err)
	}
	if err := notificationRepo.Save(ctx, notificationAgg); err != nil {
		log.Errorw("create notification failed", zap.Error(err))
		return stackErr.Error(fmt.Errorf("create notification failed: %w", err))
	}

	snapshot, err := notificationAgg.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}
	unreadCount, err := notificationRepo.CountUnread(ctx, payload.AccountID)
	if err != nil {
		return stackErr.Error(err)
	}
	if h.realtime != nil {
		if emitErr := h.realtime.EmitMessage(ctx, support.NewRealtimeNotificationPayload(notificationtypes.RealtimeEventNotificationUpsert, snapshot, unreadCount)); emitErr != nil {
			return stackErr.Error(fmt.Errorf("emit account notification realtime failed: %w", emitErr))
		}
	}
	if h.push != nil {
		if pushErr := h.push.SendNotification(ctx, snapshot); pushErr != nil {
			log.Warnw("send account notification webpush failed", zap.Error(pushErr))
		}
	}

	return stackErr.Error(h.email.SendTemplate(ctx, payload.Email, subject, "welcome.html", map[string]string{
		"Email": payload.Email,
	}))
}
