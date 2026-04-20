package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	roomprojection "wechat-clone/core/modules/room/application/projection"
	"wechat-clone/core/modules/room/domain/aggregate"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/modules/room/infra/persistent/models"
	sharedevents "wechat-clone/core/shared/contracts/events"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

type pendingRoomOutboxEvent struct {
	EventName string
	Payload   interface{}
	CreatedAt time.Time
}

type projectionSenderIdentity struct {
	Name  string
	Email string
}

type messageProjectionPayload struct {
	Message *entity.MessageEntity
	Room    *entity.Room
	Sender  projectionSenderIdentity
}

type messageAggregateProjectionPayload struct {
	Message   *entity.MessageEntity
	Room      *entity.Room
	Sender    projectionSenderIdentity
	Members   []*entity.RoomMemberEntity
	Receipts  []aggregate.PendingMessageReceipt
	Deletions []*aggregate.PendingMessageDeletion
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
	batch := make([]eventpkg.Event, 0, len(events))
	for _, pendingEvent := range events {
		nextVersion++
		batch = append(batch, eventpkg.Event{
			AggregateID:   roomID,
			AggregateType: roomOutboxAggregateType,
			Version:       nextVersion,
			EventName:     pendingEvent.EventName,
			EventData:     pendingEvent.Payload,
			CreatedAt:     pendingEvent.CreatedAt.Unix(),
		})
	}

	type batchAppender interface {
		AppendMany(ctx context.Context, events []eventpkg.Event) error
	}

	if repo, ok := outboxRepo.(batchAppender); ok {
		if err := repo.AppendMany(ctx, batch); err != nil {
			return baseVersion, stackErr.Error(fmt.Errorf("append room outbox events failed: %w", err))
		}
		return nextVersion, nil
	}

	for idx, evt := range batch {
		if err := outboxRepo.Append(ctx, evt); err != nil {
			return baseVersion, stackErr.Error(fmt.Errorf("append room outbox event #%d failed: %w", idx, err))
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

func buildMessageAggregateProjectionSyncEvent(payload messageAggregateProjectionPayload) pendingRoomOutboxEvent {
	roomID := ""
	if payload.Room != nil {
		roomID = strings.TrimSpace(payload.Room.ID)
	}

	return pendingRoomOutboxEvent{
		EventName: roomprojection.EventMessageAggregateProjectionSynced,
		Payload: &roomprojection.MessageAggregateSync{
			Message: buildMessageProjection(messageProjectionPayload{
				Message: payload.Message,
				Room:    payload.Room,
				Sender:  payload.Sender,
			}),
			Members:  mapRoomMemberProjections(payload.Members),
			Receipts: mapMessageReceiptProjections(roomID, payload.Receipts),
			Deletions: mapMessageDeletionProjections(
				roomID,
				payload.Message,
				payload.Deletions,
			),
		},
		CreatedAt: resolveMessageProjectionCreatedAt(payload),
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

func buildMessageProjection(payload messageProjectionPayload) *roomprojection.MessageProjection {
	if payload.Message == nil || payload.Room == nil {
		return nil
	}

	return &roomprojection.MessageProjection{
		RoomID:                 payload.Room.ID,
		RoomName:               payload.Room.Name,
		RoomType:               string(payload.Room.RoomType),
		MessageID:              payload.Message.ID,
		MessageContent:         payload.Message.Message,
		MessageType:            payload.Message.MessageType,
		ReplyToMessageID:       payload.Message.ReplyToMessageID,
		ForwardedFromMessageID: payload.Message.ForwardedFromMessageID,
		FileName:               payload.Message.FileName,
		FileSize:               payload.Message.FileSize,
		MimeType:               payload.Message.MimeType,
		ObjectKey:              payload.Message.ObjectKey,
		MessageSenderID:        payload.Message.SenderID,
		MessageSenderName:      strings.TrimSpace(payload.Sender.Name),
		MessageSenderEmail:     strings.TrimSpace(payload.Sender.Email),
		MessageSentAt:          payload.Message.CreatedAt.UTC(),
		Mentions:               mapProjectionMentions(payload.Message.Mentions),
		Reactions:              mapProjectionReactions(payload.Message.Reactions),
		MentionAll:             payload.Message.MentionAll,
		MentionedAccountIDs:    mapMentionedAccountIDs(payload.Message.Mentions),
		EditedAt:               cloneProjectionTime(payload.Message.EditedAt),
		DeletedForEveryoneAt:   cloneProjectionTime(payload.Message.DeletedForEveryoneAt),
	}
}

func mapProjectionReactions(items []entity.MessageReaction) []roomprojection.ProjectionReaction {
	if len(items) == 0 {
		return nil
	}

	return lo.Map(items, func(item entity.MessageReaction, _ int) roomprojection.ProjectionReaction {
		return roomprojection.ProjectionReaction{
			AccountID: strings.TrimSpace(item.AccountID),
			Emoji:     strings.TrimSpace(item.Emoji),
			ReactedAt: item.ReactedAt.UTC(),
		}
	})
}

func mapProjectionMentions(mentions []entity.MessageMention) []roomprojection.ProjectionMention {
	if len(mentions) == 0 {
		return nil
	}

	return lo.Map(mentions, func(mention entity.MessageMention, _ int) roomprojection.ProjectionMention {
		return roomprojection.ProjectionMention{
			AccountID:   mention.AccountID,
			DisplayName: mention.DisplayName,
			Username:    mention.Username,
		}
	})
}

func mapMentionedAccountIDs(mentions []entity.MessageMention) []string {
	if len(mentions) == 0 {
		return nil
	}

	results := lo.Uniq(lo.FilterMap(mentions, func(mention entity.MessageMention, _ int) (string, bool) {
		accountID := strings.TrimSpace(mention.AccountID)
		return accountID, accountID != ""
	}))
	if len(results) == 0 {
		return nil
	}
	return results
}

func mapRoomMemberProjections(members []*entity.RoomMemberEntity) []roomprojection.RoomMemberProjection {
	if len(members) == 0 {
		return nil
	}

	results := lo.FilterMap(members, func(member *entity.RoomMemberEntity, _ int) (roomprojection.RoomMemberProjection, bool) {
		if member == nil {
			return roomprojection.RoomMemberProjection{}, false
		}

		return roomprojection.RoomMemberProjection{
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
		}, true
	})
	if len(results) == 0 {
		return nil
	}
	return results
}

func mapMessageReceiptProjections(roomID string, receipts []aggregate.PendingMessageReceipt) []roomprojection.MessageReceiptProjection {
	if len(receipts) == 0 {
		return nil
	}

	return lo.Map(receipts, func(receipt aggregate.PendingMessageReceipt, _ int) roomprojection.MessageReceiptProjection {
		return roomprojection.MessageReceiptProjection{
			RoomID:      roomID,
			MessageID:   receipt.MessageID,
			AccountID:   receipt.AccountID,
			Status:      receipt.Status,
			DeliveredAt: cloneProjectionTime(receipt.DeliveredAt),
			SeenAt:      cloneProjectionTime(receipt.SeenAt),
			CreatedAt:   receipt.CreatedAt.UTC(),
			UpdatedAt:   receipt.UpdatedAt.UTC(),
		}
	})
}

func mapMessageDeletionProjections(roomID string, message *entity.MessageEntity, deletions []*aggregate.PendingMessageDeletion) []roomprojection.MessageDeletionProjection {
	if len(deletions) == 0 || message == nil {
		return nil
	}

	results := lo.FilterMap(deletions, func(deletion *aggregate.PendingMessageDeletion, _ int) (roomprojection.MessageDeletionProjection, bool) {
		if deletion == nil {
			return roomprojection.MessageDeletionProjection{}, false
		}

		return roomprojection.MessageDeletionProjection{
			RoomID:        roomID,
			MessageID:     deletion.MessageID,
			AccountID:     deletion.AccountID,
			MessageSentAt: message.CreatedAt.UTC(),
			CreatedAt:     deletion.CreatedAt.UTC(),
		}, true
	})
	if len(results) == 0 {
		return nil
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

	results := lo.FilterMap(members, func(member *entity.RoomMemberEntity, _ int) (*entity.RoomMemberEntity, bool) {
		if member == nil {
			return nil, false
		}

		copyMember := *member
		return &copyMember, true
	})
	accountIDs := lo.Uniq(lo.FilterMap(results, func(member *entity.RoomMemberEntity, _ int) (string, bool) {
		accountID := strings.TrimSpace(member.AccountID)
		return accountID, accountID != ""
	}))

	if accountRepo == nil || len(accountIDs) == 0 {
		return results, nil
	}

	accountProjections, err := accountRepo.ListByAccountIDs(ctx, accountIDs)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountMap := lo.SliceToMap(lo.Filter(accountProjections, func(account *entity.AccountEntity, _ int) bool {
		return account != nil
	}), func(account *entity.AccountEntity) (string, *entity.AccountEntity) {
		return strings.TrimSpace(account.AccountID), account
	})

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
	pendingEvent, found := lo.Find(pendingEvents, func(pendingEvent aggregate.PendingRoomOutboxEvent) bool {
		if pendingEvent.EventName != sharedevents.EventRoomMessageCreated {
			return false
		}

		payload, ok := pendingEvent.Payload.(*sharedevents.RoomMessageCreatedEvent)
		return ok && payload != nil && strings.TrimSpace(payload.MessageID) == strings.TrimSpace(messageID)
	})
	if !found {
		return "", ""
	}

	payload, _ := pendingEvent.Payload.(*sharedevents.RoomMessageCreatedEvent)
	return strings.TrimSpace(payload.MessageSenderName), strings.TrimSpace(payload.MessageSenderEmail)
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

func resolveMessageProjectionCreatedAt(payload messageAggregateProjectionPayload) time.Time {
	createdAt := payload.Message.CreatedAt.UTC()
	if payload.Message.EditedAt != nil && payload.Message.EditedAt.UTC().After(createdAt) {
		createdAt = payload.Message.EditedAt.UTC()
	}
	if payload.Message.DeletedForEveryoneAt != nil && payload.Message.DeletedForEveryoneAt.UTC().After(createdAt) {
		createdAt = payload.Message.DeletedForEveryoneAt.UTC()
	}

	for _, member := range payload.Members {
		if member != nil && member.UpdatedAt.UTC().After(createdAt) {
			createdAt = member.UpdatedAt.UTC()
		}
	}
	for _, receipt := range payload.Receipts {
		if receipt.UpdatedAt.UTC().After(createdAt) {
			createdAt = receipt.UpdatedAt.UTC()
		}
	}
	for _, deletion := range payload.Deletions {
		if deletion != nil && deletion.CreatedAt.UTC().After(createdAt) {
			createdAt = deletion.CreatedAt.UTC()
		}
	}

	return createdAt
}

func cloneProjectionTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}
