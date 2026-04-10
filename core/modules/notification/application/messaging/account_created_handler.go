package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"go-socket/core/modules/notification/domain/entity"
	"go-socket/core/modules/notification/types"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
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

	notification := &entity.NotificationEntity{
		ID:        uuid.New().String(),
		AccountID: payload.AccountID,
		Type:      types.NotificationTypeAccountCreated,
		Subject:   subject,
		Body:      body,
		CreatedAt: payload.CreatedAt,
	}
	if err := h.notificationRepo.CreateNotification(ctx, notification); err != nil {
		log.Errorw("create notification failed", zap.Error(err))
		return stackErr.Error(fmt.Errorf("create notification failed: %v", err))
	}

	return h.emailSender.Send(ctx, payload.Email, subject, body)
}
