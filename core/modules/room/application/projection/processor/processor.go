package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	roomprojectionevent "go-socket/core/modules/room/application/projection/projectionevent"
	"go-socket/core/shared/config"
	"go-socket/core/shared/contracts"
	"go-socket/core/shared/contracts/events"
	sharedevents "go-socket/core/shared/contracts/events"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

type Processor interface {
	Start() error
	Stop() error
}

type processor struct {
	consumer          []infraMessaging.Consumer
	timelineProjector events.TimelineProjector
	searchIndexer     events.MessageSearchIndexer
}

func NewProcessor(cfg *config.Config, timelineProjector events.TimelineProjector, searchIndexer events.MessageSearchIndexer) (Processor, error) {
	instance := &processor{
		consumer:          make([]infraMessaging.Consumer, 0, 1),
		timelineProjector: timelineProjector,
		searchIndexer:     searchIndexer,
	}

	topic := strings.TrimSpace(cfg.KafkaConfig.KafkaRoomConsumer.RoomOutboxTopic)
	if topic == "" || (timelineProjector == nil && searchIndexer == nil) {
		return instance, nil
	}

	handlerName := fmt.Sprintf("room-projection-%s-handler", strings.ToLower(topic))
	consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
		Servers:      cfg.KafkaConfig.KafkaServers,
		Group:        cfg.KafkaConfig.KafkaRoomConsumer.RoomGroup,
		OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
		ConsumeTopic: []string{topic},
		HandlerName:  handlerName,
		DLQ:          true,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	consumer.SetHandler(func(ctx context.Context, value []byte) error {
		return instance.handleRoomOutboxEvent(ctx, value)
	})
	instance.consumer = append(instance.consumer, consumer)

	return instance, nil
}

func (p *processor) Start() error {
	for _, consumer := range p.consumer {
		consumer.Read(p.processMessage(consumer))
	}
	return nil
}

func (p *processor) Stop() error {
	for _, consumer := range p.consumer {
		consumer.Stop()
	}
	return nil
}

func (p *processor) handleRoomOutboxEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("RoomProjection")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal room outbox event failed: %v", err))
	}

	log.Infow("handle room outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case roomprojectionevent.EventRoomProjectionUpserted:
		return p.projectRoomUpserted(ctx, event.EventData)
	case roomprojectionevent.EventRoomProjectionDeleted:
		return p.projectRoomDeleted(ctx, event.EventData)
	case roomprojectionevent.EventRoomMemberProjectionUpserted:
		return p.projectRoomMemberUpserted(ctx, event.EventData)
	case roomprojectionevent.EventRoomMemberProjectionDeleted:
		return p.projectRoomMemberDeleted(ctx, event.EventData)
	case roomprojectionevent.EventRoomMessageProjectionUpserted:
		return p.projectRoomMessageUpserted(ctx, event.EventData)
	case roomprojectionevent.EventRoomMessageReceiptProjectionUpserted:
		return p.projectRoomMessageReceiptUpserted(ctx, event.EventData)
	case roomprojectionevent.EventRoomMessageDeletionProjectionUpserted:
		return p.projectRoomMessageDeletionUpserted(ctx, event.EventData)
	default:
		return nil
	}
}

func (p *processor) projectRoomUpserted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojectionevent.EventRoomProjectionUpserted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room projection upsert payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojectionevent.RoomUpserted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojectionevent.EventRoomProjectionUpserted))
	}

	if p.timelineProjector == nil {
		return nil
	}
	return stackErr.Error(p.timelineProjector.ProjectRoom(ctx, &events.RoomProjection{
		RoomID:                 payload.RoomID,
		Name:                   payload.Name,
		Description:            payload.Description,
		RoomType:               payload.RoomType,
		OwnerID:                payload.OwnerID,
		PinnedMessageID:        payload.PinnedMessageID,
		MemberCount:            payload.MemberCount,
		HasLastMessageSnapshot: payload.HasLastMessageSnapshot,
		LastMessageID:          payload.LastMessageID,
		LastMessageAt:          payload.LastMessageAt,
		LastMessageContent:     payload.LastMessageContent,
		LastMessageSenderID:    payload.LastMessageSenderID,
		CreatedAt:              payload.CreatedAt,
		UpdatedAt:              payload.UpdatedAt,
	}))
}

func (p *processor) projectRoomDeleted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojectionevent.EventRoomProjectionDeleted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room projection delete payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojectionevent.RoomDeleted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojectionevent.EventRoomProjectionDeleted))
	}

	if p.timelineProjector == nil {
		return nil
	}
	return stackErr.Error(p.timelineProjector.DeleteProjectedRoom(ctx, payload.RoomID))
}

func (p *processor) projectRoomMemberUpserted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojectionevent.EventRoomMemberProjectionUpserted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room member projection upsert payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojectionevent.RoomMemberUpserted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojectionevent.EventRoomMemberProjectionUpserted))
	}

	if p.timelineProjector != nil {
		return stackErr.Error(p.timelineProjector.ProjectRoomMember(ctx, &events.RoomMemberProjection{
			RoomID:          payload.RoomID,
			MemberID:        payload.MemberID,
			AccountID:       payload.AccountID,
			Role:            payload.Role,
			LastDeliveredAt: payload.LastDeliveredAt,
			LastReadAt:      payload.LastReadAt,
			CreatedAt:       payload.CreatedAt,
			UpdatedAt:       payload.UpdatedAt,
		}))
	}
	return nil
}

func (p *processor) projectRoomMemberDeleted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojectionevent.EventRoomMemberProjectionDeleted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room member projection delete payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojectionevent.RoomMemberDeleted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojectionevent.EventRoomMemberProjectionDeleted))
	}

	if p.timelineProjector == nil {
		return nil
	}
	return stackErr.Error(p.timelineProjector.DeleteProjectedRoomMember(ctx, payload.RoomID, payload.AccountID))
}

func (p *processor) projectRoomMessageUpserted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojectionevent.EventRoomMessageProjectionUpserted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room message projection upsert payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojectionevent.RoomMessageUpserted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojectionevent.EventRoomMessageProjectionUpserted))
	}

	timelineProjection := buildTimelineProjection(payload)
	searchDocument := buildSearchDocument(payload)

	if p.timelineProjector != nil {
		if err := p.timelineProjector.ProjectMessage(ctx, timelineProjection); err != nil {
			return stackErr.Error(fmt.Errorf("project cassandra message failed: %v", err))
		}
	}

	if p.searchIndexer != nil {
		if err := p.searchIndexer.UpsertMessage(ctx, searchDocument); err != nil {
			return stackErr.Error(fmt.Errorf("index elasticsearch document failed: %v", err))
		}
	}
	return nil
}

func (p *processor) projectRoomMessageReceiptUpserted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojectionevent.EventRoomMessageReceiptProjectionUpserted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room message receipt projection payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojectionevent.RoomMessageReceiptUpserted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojectionevent.EventRoomMessageReceiptProjectionUpserted))
	}

	if p.timelineProjector == nil {
		return nil
	}
	return stackErr.Error(p.timelineProjector.ProjectMessageReceipt(ctx, &events.MessageReceiptProjection{
		RoomID:      payload.RoomID,
		MessageID:   payload.MessageID,
		AccountID:   payload.AccountID,
		Status:      payload.Status,
		DeliveredAt: payload.DeliveredAt,
		SeenAt:      payload.SeenAt,
		CreatedAt:   payload.CreatedAt,
		UpdatedAt:   payload.UpdatedAt,
	}))
}

func (p *processor) projectRoomMessageDeletionUpserted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojectionevent.EventRoomMessageDeletionProjectionUpserted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room message deletion projection payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojectionevent.RoomMessageDeletionUpserted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojectionevent.EventRoomMessageDeletionProjectionUpserted))
	}

	if p.timelineProjector == nil {
		return nil
	}
	return stackErr.Error(p.timelineProjector.ProjectMessageDeletion(ctx, &events.MessageDeletionProjection{
		RoomID:        payload.RoomID,
		MessageID:     payload.MessageID,
		AccountID:     payload.AccountID,
		MessageSentAt: payload.MessageSentAt,
		CreatedAt:     payload.CreatedAt,
	}))
}

func buildTimelineProjection(payload *roomprojectionevent.RoomMessageUpserted) *events.TimelineMessageProjection {
	if payload == nil {
		return nil
	}

	return &events.TimelineMessageProjection{
		RoomID:                 payload.RoomID,
		RoomName:               payload.RoomName,
		RoomType:               payload.RoomType,
		MessageID:              payload.MessageID,
		MessageContent:         payload.MessageContent,
		MessageType:            payload.MessageType,
		ReplyToMessageID:       payload.ReplyToMessageID,
		ForwardedFromMessageID: payload.ForwardedFromMessageID,
		FileName:               payload.FileName,
		FileSize:               payload.FileSize,
		MimeType:               payload.MimeType,
		ObjectKey:              payload.ObjectKey,
		MessageSenderID:        payload.MessageSenderID,
		MessageSenderName:      payload.MessageSenderName,
		MessageSenderEmail:     payload.MessageSenderEmail,
		MessageSentAt:          payload.MessageSentAt,
		Mentions: lo.Map(payload.Mentions, func(item sharedevents.RoomMessageMention, _ int) events.ProjectionMention {
			return events.ProjectionMention{
				AccountID:   item.AccountID,
				DisplayName: item.DisplayName,
				Username:    item.Username,
			}
		}),
		MentionAll:           payload.MentionAll,
		MentionedAccountIDs:  lo.Uniq(payload.MentionedAccountIDs),
		EditedAt:             payload.EditedAt,
		DeletedForEveryoneAt: payload.DeletedForEveryoneAt,
	}
}

func buildSearchDocument(payload *roomprojectionevent.RoomMessageUpserted) *events.SearchMessageDocument {
	if payload == nil {
		return nil
	}

	return &events.SearchMessageDocument{
		RoomID:                 payload.RoomID,
		RoomName:               payload.RoomName,
		RoomType:               payload.RoomType,
		MessageID:              payload.MessageID,
		MessageContent:         payload.MessageContent,
		MessageType:            payload.MessageType,
		ReplyToMessageID:       payload.ReplyToMessageID,
		ForwardedFromMessageID: payload.ForwardedFromMessageID,
		FileName:               payload.FileName,
		FileSize:               payload.FileSize,
		MimeType:               payload.MimeType,
		ObjectKey:              payload.ObjectKey,
		MessageSenderID:        payload.MessageSenderID,
		MessageSenderName:      payload.MessageSenderName,
		MessageSenderEmail:     payload.MessageSenderEmail,
		MessageSentAt:          payload.MessageSentAt,
		Mentions: lo.Map(payload.Mentions, func(item sharedevents.RoomMessageMention, _ int) events.ProjectionMention {
			return events.ProjectionMention{
				AccountID:   item.AccountID,
				DisplayName: item.DisplayName,
				Username:    item.Username,
			}
		}),
		MentionAll:          payload.MentionAll,
		MentionedAccountIDs: lo.Uniq(payload.MentionedAccountIDs),
	}
}
