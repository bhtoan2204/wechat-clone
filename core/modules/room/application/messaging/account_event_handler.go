package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"go-socket/core/modules/room/domain/entity"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/stackErr"
)

func (h *messageHandler) handleAccountCreatedEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventAccountCreated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode event payload failed: %v", err))
	}

	payload, ok := payloadAny.(*sharedevents.AccountCreatedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventAccountCreated))
	}

	if err := h.accountRepo.ProjectAccount(ctx, &entity.AccountEntity{
		AccountID:   payload.AccountID,
		DisplayName: payload.DisplayName,
		CreatedAt:   payload.CreatedAt,
	}); err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (h *messageHandler) handleAccountUpdatedEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventAccountProfileUpdated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode event payload failed: %v", err))
	}

	payload, ok := payloadAny.(*sharedevents.AccountProfileUpdatedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventAccountProfileUpdated))
	}

	if err := h.accountRepo.ProjectAccount(ctx, &entity.AccountEntity{
		AccountID:       payload.AccountID,
		DisplayName:     payload.DisplayName,
		UpdatedAt:       payload.UpdatedAt,
		AvatarObjectKey: optionalStringValue(payload.AvatarObjectKey),
		Username:        optionalStringValue(payload.Username),
	}); err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
