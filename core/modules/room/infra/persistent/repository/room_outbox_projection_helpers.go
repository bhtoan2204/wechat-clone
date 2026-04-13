package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	roomprojection "go-socket/core/modules/room/application/projection"
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

func buildRoomAggregateProjectionSyncEvent(room *entity.Room, members []*entity.RoomMemberEntity, lastMessage *entity.MessageEntity) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojection.EventRoomAggregateProjectionSynced,
		Payload: &roomprojection.RoomAggregateSync{
			Room: &roomprojection.RoomProjection{
				RoomID:          room.ID,
				Name:            room.Name,
				Description:     room.Description,
				RoomType:        string(room.RoomType),
				OwnerID:         room.OwnerID,
				PinnedMessageID: room.PinnedMessageID,
				MemberCount:     len(members),
				LastMessage:     buildRoomLastMessageProjection(lastMessage),
				CreatedAt:       room.CreatedAt.UTC(),
				UpdatedAt:       room.UpdatedAt.UTC(),
			},
			Members: mapRoomMemberProjections(members),
		},
		CreatedAt: room.UpdatedAt.UTC(),
	}
}

func buildRoomAggregateProjectionDeleteEvent(roomID string, createdAt time.Time) pendingRoomOutboxEvent {
	return pendingRoomOutboxEvent{
		EventName: roomprojection.EventRoomAggregateProjectionDeleted,
		Payload: &roomprojection.RoomAggregateDeleted{
			RoomID: strings.TrimSpace(roomID),
		},
		CreatedAt: createdAt.UTC(),
	}
}

func buildMessageAggregateProjectionSyncEvent(
	message *entity.MessageEntity,
	room *entity.Room,
	senderName,
	senderEmail string,
	members []*entity.RoomMemberEntity,
	receipts []aggregate.PendingMessageReceipt,
	deletions []*aggregate.PendingMessageDeletion,
) pendingRoomOutboxEvent {
	createdAt := message.CreatedAt.UTC()
	if message.EditedAt != nil && message.EditedAt.UTC().After(createdAt) {
		createdAt = message.EditedAt.UTC()
	}
	if message.DeletedForEveryoneAt != nil && message.DeletedForEveryoneAt.UTC().After(createdAt) {
		createdAt = message.DeletedForEveryoneAt.UTC()
	}
	for _, member := range members {
		if member != nil && member.UpdatedAt.UTC().After(createdAt) {
			createdAt = member.UpdatedAt.UTC()
		}
	}
	for _, receipt := range receipts {
		if receipt.UpdatedAt.UTC().After(createdAt) {
			createdAt = receipt.UpdatedAt.UTC()
		}
	}
	for _, deletion := range deletions {
		if deletion != nil && deletion.CreatedAt.UTC().After(createdAt) {
			createdAt = deletion.CreatedAt.UTC()
		}
	}

	return pendingRoomOutboxEvent{
		EventName: roomprojection.EventMessageAggregateProjectionSynced,
		Payload: &roomprojection.MessageAggregateSync{
			Message:  buildMessageProjection(message, room, senderName, senderEmail),
			Members:  mapRoomMemberProjections(members),
			Receipts: mapMessageReceiptProjections(strings.TrimSpace(room.ID), receipts),
			Deletions: mapMessageDeletionProjections(
				strings.TrimSpace(room.ID),
				message,
				deletions,
			),
		},
		CreatedAt: createdAt,
	}
}

func buildRoomLastMessageProjection(message *entity.MessageEntity) *roomprojection.RoomLastMessageProjection {
	if message == nil {
		return nil
	}

	lastMessageAt := message.CreatedAt.UTC()
	return &roomprojection.RoomLastMessageProjection{
		MessageID:       message.ID,
		MessageSentAt:   &lastMessageAt,
		MessageContent:  message.Message,
		MessageSenderID: message.SenderID,
	}
}

func buildMessageProjection(message *entity.MessageEntity, room *entity.Room, senderName, senderEmail string) *roomprojection.MessageProjection {
	if message == nil || room == nil {
		return nil
	}

	return &roomprojection.MessageProjection{
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
	}
}

func mapProjectionMentions(mentions []entity.MessageMention) []roomprojection.ProjectionMention {
	if len(mentions) == 0 {
		return nil
	}

	results := make([]roomprojection.ProjectionMention, 0, len(mentions))
	for _, mention := range mentions {
		results = append(results, roomprojection.ProjectionMention{
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

func mapRoomMemberProjections(members []*entity.RoomMemberEntity) []roomprojection.RoomMemberProjection {
	if len(members) == 0 {
		return nil
	}

	results := make([]roomprojection.RoomMemberProjection, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}

		results = append(results, roomprojection.RoomMemberProjection{
			RoomID:          member.RoomID,
			MemberID:        member.ID,
			AccountID:       member.AccountID,
			DisplayName:     strings.TrimSpace(member.DisplayName),
			Username:        strings.TrimSpace(member.Username),
			AvatarObjectKey: strings.TrimSpace(member.AvatarObjectKey),
			Role:            string(member.Role),
			LastDeliveredAt: cloneProjectionTime(member.LastDeliveredAt),
			LastReadAt:      cloneProjectionTime(member.LastReadAt),
			CreatedAt:       member.CreatedAt.UTC(),
			UpdatedAt:       member.UpdatedAt.UTC(),
		})
	}
	return results
}

func mapMessageReceiptProjections(roomID string, receipts []aggregate.PendingMessageReceipt) []roomprojection.MessageReceiptProjection {
	if len(receipts) == 0 {
		return nil
	}

	results := make([]roomprojection.MessageReceiptProjection, 0, len(receipts))
	for _, receipt := range receipts {
		results = append(results, roomprojection.MessageReceiptProjection{
			RoomID:      roomID,
			MessageID:   receipt.MessageID,
			AccountID:   receipt.AccountID,
			Status:      receipt.Status,
			DeliveredAt: cloneProjectionTime(receipt.DeliveredAt),
			SeenAt:      cloneProjectionTime(receipt.SeenAt),
			CreatedAt:   receipt.CreatedAt.UTC(),
			UpdatedAt:   receipt.UpdatedAt.UTC(),
		})
	}
	return results
}

func mapMessageDeletionProjections(roomID string, message *entity.MessageEntity, deletions []*aggregate.PendingMessageDeletion) []roomprojection.MessageDeletionProjection {
	if len(deletions) == 0 || message == nil {
		return nil
	}

	results := make([]roomprojection.MessageDeletionProjection, 0, len(deletions))
	for _, deletion := range deletions {
		if deletion == nil {
			continue
		}

		results = append(results, roomprojection.MessageDeletionProjection{
			RoomID:        roomID,
			MessageID:     deletion.MessageID,
			AccountID:     deletion.AccountID,
			MessageSentAt: message.CreatedAt.UTC(),
			CreatedAt:     deletion.CreatedAt.UTC(),
		})
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

func enrichRoomMembersWithAccountProjections(
	ctx context.Context,
	accountRepo repos.RoomAccountProjectionRepository,
	members []*entity.RoomMemberEntity,
) ([]*entity.RoomMemberEntity, error) {
	if len(members) == 0 {
		return nil, nil
	}

	results := make([]*entity.RoomMemberEntity, 0, len(members))
	accountIDs := make([]string, 0, len(members))
	seenAccountIDs := make(map[string]struct{}, len(members))

	for _, member := range members {
		if member == nil {
			continue
		}

		copyMember := *member
		results = append(results, &copyMember)

		accountID := strings.TrimSpace(member.AccountID)
		if accountID == "" {
			continue
		}
		if _, exists := seenAccountIDs[accountID]; exists {
			continue
		}

		seenAccountIDs[accountID] = struct{}{}
		accountIDs = append(accountIDs, accountID)
	}

	if accountRepo == nil || len(accountIDs) == 0 {
		return results, nil
	}

	accountProjections, err := accountRepo.ListByAccountIDs(ctx, accountIDs)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountMap := make(map[string]*entity.AccountEntity, len(accountProjections))
	for _, account := range accountProjections {
		if account == nil {
			continue
		}
		accountMap[strings.TrimSpace(account.AccountID)] = account
	}

	for _, member := range results {
		if member == nil {
			continue
		}

		account, exists := accountMap[strings.TrimSpace(member.AccountID)]
		if !exists || account == nil {
			continue
		}

		if displayName := strings.TrimSpace(account.DisplayName); displayName != "" {
			member.DisplayName = displayName
		}
		if username := strings.TrimSpace(account.Username); username != "" {
			member.Username = username
		}
		if avatarObjectKey := strings.TrimSpace(account.AvatarObjectKey); avatarObjectKey != "" {
			member.AvatarObjectKey = avatarObjectKey
		}
	}

	return results, nil
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
