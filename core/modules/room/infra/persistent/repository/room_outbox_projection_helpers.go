package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	roomprojectionevent "go-socket/core/modules/room/application/projection/projectionevent"
	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"
	sharedevents "go-socket/core/shared/contracts/events"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type pendingRoomOutboxEvent struct {
	EventName string
	Payload   interface{}
	CreatedAt time.Time
}

func loadLatestRoomOutboxVersion(ctx context.Context, db *gorm.DB, roomID string) (int, error) {
	var result struct {
		Version int
	}

	err := db.WithContext(ctx).
		Model(&models.RoomOutboxEventModel{}).
		Select("COALESCE(MAX(version), 0) AS version").
		Where("aggregate_id = ?", roomID).
		Scan(&result).Error
	if err != nil {
		return 0, stackErr.Error(err)
	}
	return result.Version, nil
}

func appendRoomOutboxEvents(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID string, baseVersion int, events []pendingRoomOutboxEvent) (int, error) {
	nextVersion := baseVersion
	for idx, pendingEvent := range events {
		nextVersion++
		if err := outboxRepo.Append(ctx, eventpkg.Event{
			AggregateID:   roomID,
			AggregateType: roomOutboxAggregateType,
			Version:       nextVersion,
			EventName:     pendingEvent.EventName,
			EventData:     pendingEvent.Payload,
			CreatedAt:     pendingEvent.CreatedAt.Unix(),
		}); err != nil {
			return baseVersion, stackErr.Error(fmt.Errorf("append room outbox event #%d failed: %v", idx, err))
		}
	}
	return nextVersion, nil
}

func buildRoomProjectionUpsertEvent(room *entity.Room, memberCount int, lastMessage *entity.MessageEntity, hasLastMessageSnapshot bool) pendingRoomOutboxEvent {
	var lastMessageAt *time.Time
	lastMessageID := ""
	lastMessageContent := ""
	lastMessageSenderID := ""
	if lastMessage != nil {
		value := lastMessage.CreatedAt.UTC()
		lastMessageAt = &value
		lastMessageID = lastMessage.ID
		lastMessageContent = lastMessage.Message
		lastMessageSenderID = lastMessage.SenderID
	}

	return pendingRoomOutboxEvent{
		EventName: roomprojectionevent.EventRoomProjectionUpserted,
		Payload: &roomprojectionevent.RoomUpserted{
			RoomID:                 room.ID,
			Name:                   room.Name,
			Description:            room.Description,
			RoomType:               string(room.RoomType),
			OwnerID:                room.OwnerID,
			PinnedMessageID:        room.PinnedMessageID,
			MemberCount:            memberCount,
			HasLastMessageSnapshot: hasLastMessageSnapshot,
			LastMessageID:          lastMessageID,
			LastMessageAt:          lastMessageAt,
			LastMessageContent:     lastMessageContent,
			LastMessageSenderID:    lastMessageSenderID,
			CreatedAt:              room.CreatedAt.UTC(),
			UpdatedAt:              room.UpdatedAt.UTC(),
		},
		CreatedAt: room.UpdatedAt.UTC(),
	}
}

func buildRoomProjectionDeleteEvent(roomID string, createdAt time.Time) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojectionevent.EventRoomProjectionDeleted,
		Payload: &roomprojectionevent.RoomDeleted{
			RoomID: strings.TrimSpace(roomID),
		},
		CreatedAt: createdAt.UTC(),
	}
}

func buildRoomMemberProjectionUpsertEvent(member *entity.RoomMemberEntity) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojectionevent.EventRoomMemberProjectionUpserted,
		Payload: &roomprojectionevent.RoomMemberUpserted{
			RoomID:          member.RoomID,
			MemberID:        member.ID,
			AccountID:       member.AccountID,
			Role:            string(member.Role),
			LastDeliveredAt: member.LastDeliveredAt,
			LastReadAt:      member.LastReadAt,
			CreatedAt:       member.CreatedAt.UTC(),
			UpdatedAt:       member.UpdatedAt.UTC(),
		},
		CreatedAt: member.UpdatedAt.UTC(),
	}
}

func buildRoomMemberProjectionDeleteEvent(roomID, accountID string, createdAt time.Time) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojectionevent.EventRoomMemberProjectionDeleted,
		Payload: &roomprojectionevent.RoomMemberDeleted{
			RoomID:    strings.TrimSpace(roomID),
			AccountID: strings.TrimSpace(accountID),
		},
		CreatedAt: createdAt.UTC(),
	}
}

func buildRoomMessageProjectionUpsertEvent(message *entity.MessageEntity, room *entity.Room, senderName, senderEmail string) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojectionevent.EventRoomMessageProjectionUpserted,
		Payload: &roomprojectionevent.RoomMessageUpserted{
			RoomID:                 room.ID,
			RoomName:               room.Name,
			RoomType:               string(room.RoomType),
			MessageID:              message.ID,
			MessageContent:         message.Message,
			MessageType:            message.MessageType,
			ReplyToMessageID:       message.ReplyToMessageID,
			ForwardedFromMessageID: message.ForwardedFromMessageID,
			FileName:               message.FileName,
			FileSize:               message.FileSize,
			MimeType:               message.MimeType,
			ObjectKey:              message.ObjectKey,
			MessageSenderID:        message.SenderID,
			MessageSenderName:      strings.TrimSpace(senderName),
			MessageSenderEmail:     strings.TrimSpace(senderEmail),
			MessageSentAt:          message.CreatedAt.UTC(),
			Mentions:               mapProjectionMentions(message.Mentions),
			MentionAll:             message.MentionAll,
			MentionedAccountIDs:    mapMentionedAccountIDs(message.Mentions),
			EditedAt:               cloneProjectionTime(message.EditedAt),
			DeletedForEveryoneAt:   cloneProjectionTime(message.DeletedForEveryoneAt),
		},
		CreatedAt: message.CreatedAt.UTC(),
	}
}

func buildRoomMessageReceiptProjectionEvent(roomID string, receipt aggregate.PendingMessageReceipt) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojectionevent.EventRoomMessageReceiptProjectionUpserted,
		Payload: &roomprojectionevent.RoomMessageReceiptUpserted{
			RoomID:      strings.TrimSpace(roomID),
			MessageID:   receipt.MessageID,
			AccountID:   receipt.AccountID,
			Status:      receipt.Status,
			DeliveredAt: cloneProjectionTime(receipt.DeliveredAt),
			SeenAt:      cloneProjectionTime(receipt.SeenAt),
			CreatedAt:   receipt.CreatedAt.UTC(),
			UpdatedAt:   receipt.UpdatedAt.UTC(),
		},
		CreatedAt: receipt.UpdatedAt.UTC(),
	}
}

func buildRoomMessageDeletionProjectionEvent(roomID string, message *entity.MessageEntity, deletion *aggregate.PendingMessageDeletion) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojectionevent.EventRoomMessageDeletionProjectionUpserted,
		Payload: &roomprojectionevent.RoomMessageDeletionUpserted{
			RoomID:        strings.TrimSpace(roomID),
			MessageID:     deletion.MessageID,
			AccountID:     deletion.AccountID,
			MessageSentAt: message.CreatedAt.UTC(),
			CreatedAt:     deletion.CreatedAt.UTC(),
		},
		CreatedAt: deletion.CreatedAt.UTC(),
	}
}

func mapProjectionMentions(mentions []entity.MessageMention) []sharedevents.RoomMessageMention {
	if len(mentions) == 0 {
		return nil
	}

	results := make([]sharedevents.RoomMessageMention, 0, len(mentions))
	for _, mention := range mentions {
		results = append(results, sharedevents.RoomMessageMention{
			AccountID:   mention.AccountID,
			DisplayName: mention.DisplayName,
			Username:    mention.Username,
		})
	}
	return results
}

func mapMentionedAccountIDs(mentions []entity.MessageMention) []string {
	if len(mentions) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(mentions))
	results := make([]string, 0, len(mentions))
	for _, mention := range mentions {
		accountID := strings.TrimSpace(mention.AccountID)
		if accountID == "" {
			continue
		}
		if _, ok := seen[accountID]; ok {
			continue
		}
		seen[accountID] = struct{}{}
		results = append(results, accountID)
	}
	return results
}

func sortRoomMembersByAccount(members []*entity.RoomMemberEntity) []*entity.RoomMemberEntity {
	results := append([]*entity.RoomMemberEntity(nil), members...)
	sort.Slice(results, func(i, j int) bool {
		if results[i] == nil || results[j] == nil {
			return i < j
		}
		return strings.TrimSpace(results[i].AccountID) < strings.TrimSpace(results[j].AccountID)
	})
	return results
}

func senderIdentityFromPendingEvents(messageID string, pendingEvents []aggregate.PendingRoomOutboxEvent) (string, string) {
	for _, pendingEvent := range pendingEvents {
		if pendingEvent.EventName != sharedevents.EventRoomMessageCreated {
			continue
		}
		payload, ok := pendingEvent.Payload.(*sharedevents.RoomMessageCreatedEvent)
		if !ok || payload == nil {
			continue
		}
		if strings.TrimSpace(payload.MessageID) != strings.TrimSpace(messageID) {
			continue
		}
		return strings.TrimSpace(payload.MessageSenderName), strings.TrimSpace(payload.MessageSenderEmail)
	}
	return "", ""
}

func senderIdentityFromProjection(account *entity.AccountEntity, fallback string) (string, string) {
	fallback = strings.TrimSpace(fallback)
	if account == nil {
		return fallback, ""
	}

	switch {
	case strings.TrimSpace(account.DisplayName) != "":
		return strings.TrimSpace(account.DisplayName), ""
	case strings.TrimSpace(account.Username) != "":
		return strings.TrimSpace(account.Username), ""
	default:
		return fallback, ""
	}
}

func cloneProjectionTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}
