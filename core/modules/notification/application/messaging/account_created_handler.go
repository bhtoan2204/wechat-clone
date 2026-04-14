package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-socket/core/modules/notification/domain/aggregate"
	"go-socket/core/modules/notification/domain/repos"
	"go-socket/core/modules/notification/types"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (h *messageHandler) handleAccountCreatedEvent(ctx context.Context, raw json.RawMessage) error {
	log := logging.FromContext(ctx).Named("handleAccountCreatedEvent")
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventAccountCreated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode event payload failed: %v", err))
	}

	payload, ok := payloadAny.(*sharedevents.AccountCreatedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventAccountCreated))
	}

	subject := "Welcome to Go Socket"
	body := fmt.Sprintf("Welcome %s!", payload.Email)
	notificationID := aggregate.WelcomeNotificationID(payload.AccountID)

	if _, err := h.notificationRepo.Load(ctx, notificationID); err == nil {
		return nil
	} else if !errors.Is(err, repos.ErrNotificationNotFound) {
		log.Errorw("load notification failed", zap.Error(err))
		return stackErr.Error(fmt.Errorf("load notification failed: %v", err))
	}

	notificationAgg, err := aggregate.NewNotificationAggregate(notificationID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := notificationAgg.Create(
		payload.AccountID,
		types.NotificationTypeAccountCreated,
		subject,
		body,
		payload.CreatedAt,
	); err != nil {
		return stackErr.Error(err)
	}
	if err := h.notificationRepo.Save(ctx, notificationAgg); err != nil {
		log.Errorw("create notification failed", zap.Error(err))
		return stackErr.Error(fmt.Errorf("create notification failed: %v", err))
	}

	return h.emailSender.Send(ctx, payload.Email, subject, body)
}
