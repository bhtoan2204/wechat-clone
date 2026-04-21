package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"wechat-clone/core/modules/notification/application/support"
	"wechat-clone/core/modules/notification/domain/aggregate"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	notificationtypes "wechat-clone/core/modules/notification/types"
	roomprojection "wechat-clone/core/modules/room/application/projection"
	"wechat-clone/core/shared/contracts"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (h *messageHandler) handleRoomOutboxEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("handleRoomOutboxEvent")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal room outbox event failed: %w", err))
	}

	log.Infow("handle room outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case sharedevents.EventRoomMessageCreated:
		return h.handleRoomMentionNotificationEvent(ctx, event.EventData)
	case roomprojection.EventMessageAggregateProjectionSynced:
		return h.handleRoomMessageProjectionEvent(ctx, event.EventData)
	default:
		return nil
	}
}

func (h *messageHandler) handleRoomMentionNotificationEvent(ctx context.Context, raw json.RawMessage) error {
	log := logging.FromContext(ctx).Named("handleRoomMentionNotificationEvent")
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventRoomMessageCreated, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room message created payload failed: %w", err))
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

		notificationAgg, err := aggregate.NewNotificationAggregate(
			aggregate.RoomMentionNotificationID(payload.MessageID, accountID),
		)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := notificationAgg.Create(
			accountID,
			notificationtypes.NotificationTypeRoomMention,
			buildRoomMentionSubject(payload),
			buildRoomMentionBody(payload),
			payload.MessageSentAt,
		); err != nil {
			return stackErr.Error(err)
		}
		if err := h.baseRepo.NotificationRepository().Save(ctx, notificationAgg); err != nil {
			return stackErr.Error(fmt.Errorf("create room mention notification failed: %w", err))
		}

		snapshot, err := notificationAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}
		unreadCount, err := h.baseRepo.NotificationRepository().CountUnread(ctx, accountID)
		if err != nil {
			return stackErr.Error(err)
		}
		if emitErr := h.realtime.EmitMessage(ctx, support.NewRealtimeNotificationPayload(notificationtypes.RealtimeEventNotificationUpsert, snapshot, unreadCount)); emitErr != nil {
			log.Warnw("emit room mention notification realtime failed", zap.Error(emitErr))
		}

		if pushErr := h.push.SendNotification(ctx, snapshot); pushErr != nil {
			log.Warnw("send room mention webpush failed", zap.Error(pushErr))
		}
	}

	return nil
}

func (h *messageHandler) handleRoomMessageProjectionEvent(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojection.EventMessageAggregateProjectionSynced, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room message projection payload failed: %w", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojection.MessageAggregateSync)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojection.EventMessageAggregateProjectionSynced))
	}
	if payload.Message == nil {
		return nil
	}
	if payload.Message.DeletedForEveryoneAt != nil {
		return nil
	}

	for _, member := range payload.Members {
		accountID := strings.TrimSpace(member.AccountID)
		if accountID == "" || accountID == strings.TrimSpace(payload.Message.MessageSenderID) {
			continue
		}

		groupKey := roomMessageGroupKey(payload.Message.RoomID)
		notificationRepo := h.baseRepo.NotificationRepository()
		notificationAgg, err := notificationRepo.LoadMessageGroup(ctx, accountID, groupKey)
		if err != nil && !errors.Is(err, notificationrepos.ErrNotificationNotFound) {
			return stackErr.Error(err)
		}

		input := aggregate.MessageNotificationInput{
			AccountID:      accountID,
			GroupKey:       groupKey,
			Subject:        buildRoomMessageSubject(payload.Message),
			Body:           buildRoomMessageBody(payload.Message),
			RoomID:         strings.TrimSpace(payload.Message.RoomID),
			RoomName:       strings.TrimSpace(payload.Message.RoomName),
			SenderID:       strings.TrimSpace(payload.Message.MessageSenderID),
			SenderName:     resolveProjectionSenderName(payload.Message),
			MessageID:      strings.TrimSpace(payload.Message.MessageID),
			MessagePreview: buildRoomMessagePreview(payload.Message),
			MessageAt:      payload.Message.MessageSentAt,
		}

		if notificationAgg == nil {
			notificationAgg, err = aggregate.NewNotificationAggregate(aggregate.RoomMessageNotificationID(accountID, groupKey))
			if err != nil {
				return stackErr.Error(err)
			}
			if err := notificationAgg.CreateMessageNotification(input); err != nil {
				return stackErr.Error(err)
			}
		} else {
			changed, err := notificationAgg.ApplyMessageActivity(input)
			if err != nil {
				return stackErr.Error(err)
			}
			if !changed {
				continue
			}
		}

		if err := notificationRepo.Save(ctx, notificationAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save message notification failed: %w", err))
		}

		snapshot, err := notificationAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}
		unreadCount, err := notificationRepo.CountUnread(ctx, accountID)
		if err != nil {
			return stackErr.Error(err)
		}
		if h.realtime != nil {
			if emitErr := h.realtime.EmitMessage(ctx, support.NewRealtimeNotificationPayload(notificationtypes.RealtimeEventNotificationUpsert, snapshot, unreadCount)); emitErr != nil {
				return stackErr.Error(fmt.Errorf("emit message notification realtime failed: %w", emitErr))
			}
		}
		if h.push != nil {
			if pushErr := h.push.SendNotification(ctx, snapshot); pushErr != nil {
				logging.FromContext(ctx).Warnw("send message notification webpush failed", zap.Error(pushErr))
			}
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

func buildRoomMentionSubject(payload *sharedevents.RoomMessageCreatedEvent) string {
	senderName := resolveRoomSenderName(payload)
	roomName := resolveRoomName(payload)
	if payload != nil && payload.MentionAll {
		return fmt.Sprintf("%s mentioned everyone in %s", senderName, roomName)
	}
	return fmt.Sprintf("%s mentioned you in %s", senderName, roomName)
}

func buildRoomMentionBody(payload *sharedevents.RoomMessageCreatedEvent) string {
	if payload == nil {
		return "You were mentioned in a conversation"
	}
	return buildRawMessagePreview(payload.MessageType, payload.MessageContent, payload.FileName)
}

func buildRoomMessageSubject(message *roomprojection.MessageProjection) string {
	if message == nil {
		return "New message"
	}
	roomName := strings.TrimSpace(message.RoomName)
	if roomName == "" {
		roomName = strings.TrimSpace(message.RoomID)
	}
	if roomName == "" {
		roomName = "a conversation"
	}
	return fmt.Sprintf("%s sent a message in %s", resolveProjectionSenderName(message), roomName)
}

func buildRoomMessageBody(message *roomprojection.MessageProjection) string {
	if message == nil {
		return "You have a new message"
	}
	return buildRawMessagePreview(message.MessageType, message.MessageContent, message.FileName)
}

func buildRoomMessagePreview(message *roomprojection.MessageProjection) string {
	if message == nil {
		return ""
	}
	return buildRawMessagePreview(message.MessageType, message.MessageContent, message.FileName)
}

func buildRawMessagePreview(messageType, content, fileName string) string {
	content = strings.TrimSpace(content)
	if content != "" {
		if len(content) > 180 {
			return content[:177] + "..."
		}
		return content
	}

	switch strings.ToLower(strings.TrimSpace(messageType)) {
	case "image":
		return "Sent an image"
	case "file":
		if strings.TrimSpace(fileName) != "" {
			return "Sent a file: " + strings.TrimSpace(fileName)
		}
		return "Sent a file"
	case "transfer":
		return "Sent a transfer"
	default:
		return "Sent a message"
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

func resolveProjectionSenderName(message *roomprojection.MessageProjection) string {
	if message == nil {
		return "Someone"
	}
	switch {
	case strings.TrimSpace(message.MessageSenderName) != "":
		return strings.TrimSpace(message.MessageSenderName)
	case strings.TrimSpace(message.MessageSenderID) != "":
		return strings.TrimSpace(message.MessageSenderID)
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

func roomMessageGroupKey(roomID string) string {
	return notificationtypes.MessageNotificationGroupPrefix + strings.TrimSpace(roomID)
}
