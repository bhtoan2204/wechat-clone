package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	roomprojection "go-socket/core/modules/room/application/projection"
	"go-socket/core/shared/config"
	"go-socket/core/shared/contracts"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

//go:generate mockgen -package=processor -destination=processor_mock.go -source=processor.go
type Processor interface {
	Start() error
	Stop() error
}

type processor struct {
	consumer         []infraMessaging.Consumer
	servingProjector roomprojection.ServingProjector
	searchIndexer    roomprojection.MessageSearchIndexer
}

func NewProcessor(cfg *config.Config, servingProjector roomprojection.ServingProjector, searchIndexer roomprojection.MessageSearchIndexer) (Processor, error) {
	instance := &processor{
		consumer:         make([]infraMessaging.Consumer, 0, 1),
		servingProjector: servingProjector,
		searchIndexer:    searchIndexer,
	}

	topic := strings.TrimSpace(cfg.KafkaConfig.KafkaRoomConsumer.RoomOutboxTopic)
	if topic == "" || (servingProjector == nil && searchIndexer == nil) {
		return instance, nil
	}

	handlerName := fmt.Sprintf("room-projection-%s-handler", strings.ToLower(topic))
	consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
		Servers:      cfg.KafkaConfig.KafkaServers,
		Group:        cfg.KafkaConfig.KafkaRoomConsumer.RoomProjectionGroup,
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
	case roomprojection.EventRoomAggregateProjectionSynced:
		return p.projectRoomAggregateSynced(ctx, event.EventData)
	case roomprojection.EventRoomAggregateProjectionDeleted:
		return p.projectRoomAggregateDeleted(ctx, event.EventData)
	case roomprojection.EventMessageAggregateProjectionSynced:
		return p.projectMessageAggregateSynced(ctx, event.EventData)
	default:
		return nil
	}
}

func (p *processor) projectRoomAggregateSynced(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojection.EventRoomAggregateProjectionSynced, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room aggregate projection payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojection.RoomAggregateSync)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojection.EventRoomAggregateProjectionSynced))
	}
	if p.servingProjector == nil {
		return nil
	}
	return stackErr.Error(p.servingProjector.SyncRoomAggregate(ctx, payload))
}

func (p *processor) projectRoomAggregateDeleted(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojection.EventRoomAggregateProjectionDeleted, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode room aggregate delete payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojection.RoomAggregateDeleted)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojection.EventRoomAggregateProjectionDeleted))
	}
	if p.servingProjector == nil {
		if p.searchIndexer == nil {
			return nil
		}
		return stackErr.Error(p.searchIndexer.DeleteRoom(ctx, payload.RoomID))
	}
	if err := p.servingProjector.DeleteRoomAggregate(ctx, payload.RoomID); err != nil {
		return stackErr.Error(err)
	}
	if p.searchIndexer != nil {
		return stackErr.Error(p.searchIndexer.DeleteRoom(ctx, payload.RoomID))
	}
	return nil
}

func (p *processor) projectMessageAggregateSynced(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, roomprojection.EventMessageAggregateProjectionSynced, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode message aggregate projection payload failed: %v", err))
	}
	if payloadAny == nil {
		return nil
	}

	payload, ok := payloadAny.(*roomprojection.MessageAggregateSync)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", roomprojection.EventMessageAggregateProjectionSynced))
	}

	if p.servingProjector != nil {
		if err := p.servingProjector.SyncMessageAggregate(ctx, payload); err != nil {
			return stackErr.Error(fmt.Errorf("sync cassandra message aggregate failed: %v", err))
		}
	}

	if p.searchIndexer != nil && payload.Message != nil {
		if err := p.searchIndexer.SyncMessage(ctx, payload.Message); err != nil {
			return stackErr.Error(fmt.Errorf("sync elasticsearch message failed: %v", err))
		}
	}
	return nil
}
