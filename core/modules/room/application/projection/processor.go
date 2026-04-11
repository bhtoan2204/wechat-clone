package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-socket/core/shared/config"
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
	timelineProjector TimelineProjector
	searchIndexer     MessageSearchIndexer
}

func NewProcessor(cfg *config.Config, timelineProjector TimelineProjector, searchIndexer MessageSearchIndexer) (Processor, error) {
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

	var event roomOutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal room outbox event failed: %v", err))
	}

	log.Infow("handle room outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case sharedevents.EventRoomMessageCreated:
		return p.projectRoomMessageCreated(ctx, event.EventData)
	default:
		return nil
	}
}

func (p *processor) projectRoomMessageCreated(ctx context.Context, raw json.RawMessage) error {
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

	timelineProjection := buildTimelineProjection(payload)
	searchDocument := buildSearchDocument(payload)

	if p.timelineProjector != nil {
		if err := p.timelineProjector.UpsertMessage(ctx, timelineProjection); err != nil {
			return stackErr.Error(fmt.Errorf("project cassandra timeline failed: %v", err))
		}
	}

	if p.searchIndexer != nil {
		if err := p.searchIndexer.UpsertMessage(ctx, searchDocument); err != nil {
			return stackErr.Error(fmt.Errorf("index elasticsearch document failed: %v", err))
		}
	}

	return nil
}

func buildTimelineProjection(payload *sharedevents.RoomMessageCreatedEvent) *TimelineMessageProjection {
	if payload == nil {
		return nil
	}

	return &TimelineMessageProjection{
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
		Mentions: lo.Map(payload.Mentions, func(item sharedevents.RoomMessageMention, _ int) ProjectionMention {
			return ProjectionMention{
				AccountID:   item.AccountID,
				DisplayName: item.DisplayName,
				Username:    item.Username,
			}
		}),
		MentionAll:          payload.MentionAll,
		MentionedAccountIDs: lo.Uniq(payload.MentionedAccountIDs),
	}
}

func buildSearchDocument(payload *sharedevents.RoomMessageCreatedEvent) *SearchMessageDocument {
	if payload == nil {
		return nil
	}

	return &SearchMessageDocument{
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
		Mentions: lo.Map(payload.Mentions, func(item sharedevents.RoomMessageMention, _ int) ProjectionMention {
			return ProjectionMention{
				AccountID:   item.AccountID,
				DisplayName: item.DisplayName,
				Username:    item.Username,
			}
		}),
		MentionAll:          payload.MentionAll,
		MentionedAccountIDs: lo.Uniq(payload.MentionedAccountIDs),
	}
}
