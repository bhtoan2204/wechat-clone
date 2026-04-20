package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"wechat-clone/core/modules/relationship/domain/entity"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/stackErr"
)

func (h *messageHandler) handleAccountCreatedEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventAccountCreated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode event payload failed: %w", err))
	}

	payload, ok := payloadAny.(*sharedevents.AccountCreatedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventAccountCreated))
	}

	return stackErr.Error(h.accountRepo.ProjectAccount(ctx, &entity.AccountProjection{
		AccountID:   payload.AccountID,
		DisplayName: resolveAccountCreatedDisplayName(payload),
		CreatedAt:   payload.CreatedAt,
		UpdatedAt:   payload.CreatedAt,
	}))
}

func (h *messageHandler) handleAccountUpdatedEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventAccountProfileUpdated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode event payload failed: %w", err))
	}

	payload, ok := payloadAny.(*sharedevents.AccountProfileUpdatedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventAccountProfileUpdated))
	}

	account, err := h.accountRepo.GetByID(ctx, payload.AccountID)
	if err != nil {
		// If the create event is still lagging, build from the update payload so the projection remains self-healing.
		account = &entity.AccountProjection{
			AccountID: payload.AccountID,
		}
	}

	account.DisplayName = payload.DisplayName
	account.Username = strings.TrimSpace(valueOrEmpty(payload.Username))
	account.AvatarObjectKey = strings.TrimSpace(valueOrEmpty(payload.AvatarObjectKey))
	account.UpdatedAt = payload.UpdatedAt

	return stackErr.Error(h.accountRepo.ProjectAccount(ctx, account))
}

func resolveAccountCreatedDisplayName(payload *sharedevents.AccountCreatedEvent) string {
	if payload == nil {
		return ""
	}

	if displayName := strings.TrimSpace(payload.DisplayName); displayName != "" {
		return displayName
	}
	if email := strings.TrimSpace(payload.Email); email != "" {
		return email
	}
	return strings.TrimSpace(payload.AccountID)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
