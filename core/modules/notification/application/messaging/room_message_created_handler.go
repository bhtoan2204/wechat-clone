package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-socket/core/modules/notification/domain/entity"
	"go-socket/core/modules/notification/types"
	"go-socket/core/shared/contracts"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (h *messageHandler) handleRoomOutboxEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("handleRoomOutboxEvent")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal room outbox event failed: %v", err))
	}

	log.Infow("handle room outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case sharedevents.EventRoomMessageCreated:
		return h.handleRoomMessageCreatedEvent(ctx, event.EventData)
	default:
		return nil
	}
}

func (h *messageHandler) handleRoomMessageCreatedEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventRoomMessageCreated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room message created payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*sharedevents.RoomMessageCreatedEvent)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventRoomMessageCreated))
	}

	recipients := normalizeMentionRecipients(payload)
	for _, accountID := range recipients {
		if accountID == "" || accountID == payload.MessageSenderID {
			continue
		}

		notification := &entity.NotificationEntity{
			ID:        mentionNotificationID(payload.MessageID, accountID),
			AccountID: accountID,
			Type:      types.NotificationTypeRoomMention,
			Subject:   buildRoomMentionSubject(payload),
			Body:      buildRoomMentionBody(payload),
			CreatedAt: payload.MessageSentAt,
		}
		if err := h.notificationRepo.CreateNotification(ctx, notification); err != nil {
			return stackErr.Error(fmt.Errorf("create room mention notification failed: %v", err))
		}
	}

	return nil
}

func normalizeMentionRecipients(payload *sharedevents.RoomMessageCreatedEvent) []string {
	if payload == nil || len(payload.MentionedAccountIDs) == 0 {
		return nil
	}

	recipients := make([]string, 0, len(payload.MentionedAccountIDs))
	seen := make(map[string]struct{}, len(payload.MentionedAccountIDs))
	for _, item := range payload.MentionedAccountIDs {
		accountID := strings.TrimSpace(item)
		if accountID == "" {
			continue
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		recipients = append(recipients, accountID)
	}
	return recipients
}

func mentionNotificationID(messageID, accountID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("room-message-mention:"+strings.TrimSpace(messageID)+":"+strings.TrimSpace(accountID))).String()
}

func buildRoomMentionSubject(payload *sharedevents.RoomMessageCreatedEvent) string {
	senderName := resolveRoomSenderName(payload)
	roomName := resolveRoomName(payload)
	if payload != nil && payload.MentionAll {
		return fmt.Sprintf("%s mentioned everyone in %s", senderName, roomName)
	}
	return fmt.Sprintf("%s mentioned you in %s", senderName, roomName)
}

func buildRoomMentionBody(payload *sharedevents.RoomMessageCreatedEvent) string {
	senderName := resolveRoomSenderName(payload)
	if payload == nil {
		return senderName + " mentioned you"
	}

	content := strings.TrimSpace(payload.MessageContent)
	if content != "" {
		if len(content) > 180 {
			content = content[:177] + "..."
		}
		return content
	}

	switch strings.ToLower(strings.TrimSpace(payload.MessageType)) {
	case "image":
		return senderName + " sent an image"
	case "file":
		return senderName + " sent a file"
	default:
		return senderName + " mentioned you"
	}
}

func resolveRoomSenderName(payload *sharedevents.RoomMessageCreatedEvent) string {
	if payload == nil {
		return "Someone"
	}
	switch {
	case strings.TrimSpace(payload.MessageSenderName) != "":
		return strings.TrimSpace(payload.MessageSenderName)
	case strings.TrimSpace(payload.MessageSenderID) != "":
		return strings.TrimSpace(payload.MessageSenderID)
	default:
		return "Someone"
	}
}

func resolveRoomName(payload *sharedevents.RoomMessageCreatedEvent) string {
	if payload == nil {
		return "a group chat"
	}
	switch {
	case strings.TrimSpace(payload.RoomName) != "":
		return strings.TrimSpace(payload.RoomName)
	case strings.TrimSpace(payload.RoomID) != "":
		return strings.TrimSpace(payload.RoomID)
	default:
		return "a group chat"
	}
}
