package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"go-socket/core/modules/room/application/service"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/config"
	"go-socket/core/shared/contracts"
	sharedevents "go-socket/core/shared/contracts/events"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"strings"

	"go.uber.org/zap"
)

//go:generate mockgen -package=messaging -destination=message_handler_mock.go -source=message_handler.go
type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer    []infraMessaging.Consumer
	accountRepo repos.RoomAccountProjectionRepository
	baseRepo    repos.Repos
	svc         service.RealtimeService
}

func NewMessageHandler(cfg *config.Config, repos repos.Repos, svc service.RealtimeService) (MessageHandler, error) {
	instance := &messageHandler{
		consumer:    make([]infraMessaging.Consumer, 0),
		accountRepo: repos.RoomAccountProjectionRepository(),
		baseRepo:    repos,
		svc:         svc,
	}

	consumeTopics := []string{
		cfg.KafkaConfig.KafkaRoomConsumer.AccountTopic,
		cfg.KafkaConfig.KafkaRoomConsumer.LedgerOutboxTopic,
	}
	mapHandler := map[string]infraMessaging.Handler{
		fmt.Sprintf("room-%s-handler", strings.ToLower(cfg.KafkaConfig.KafkaRoomConsumer.AccountTopic)): func(ctx context.Context, value []byte) error {
			return instance.handleAccountEvent(ctx, value)
		},
		fmt.Sprintf("room-%s-handler", strings.ToLower(cfg.KafkaConfig.KafkaRoomConsumer.LedgerOutboxTopic)): func(ctx context.Context, value []byte) error {
			return instance.handleLedgerEvent(ctx, value)
		},
	}

	for _, topic := range consumeTopics {
		consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
			Servers:      cfg.KafkaConfig.KafkaServers,
			Group:        cfg.KafkaConfig.KafkaRoomConsumer.RoomMessagingGroup,
			OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
			ConsumeTopic: []string{topic},
			HandlerName:  fmt.Sprintf("room-%s-handler", strings.ToLower(topic)),
			DLQ:          true,
		})
		if err != nil {
			return nil, stackErr.Error(err)
		}
		consumer.SetHandler(mapHandler[fmt.Sprintf("room-%s-handler", strings.ToLower(topic))])
		instance.consumer = append(instance.consumer, consumer)
	}

	return instance, nil
}

func (h *messageHandler) Start() error {
	for _, consumer := range h.consumer {
		consumer.Read(infraMessaging.WrapConsumerCallback(consumer, "Handle room message failed"))
	}
	return nil
}

func (h *messageHandler) Stop() error {
	infraMessaging.StopConsumers(h.consumer)
	return nil
}

func (h *messageHandler) handleAccountEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("handleAccountEvent")
	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal account outbox event failed: %v", err))
	}
	log.Infow("handle account event", zap.String("event_name", event.EventName))
	switch event.EventName {
	case sharedevents.EventAccountCreated:
		if err := h.handleAccountCreatedEvent(ctx, event.EventData); err != nil {
			return stackErr.Error(err)
		}
	case sharedevents.EventAccountProfileUpdated:
		if err := h.handleAccountUpdatedEvent(ctx, event.EventData); err != nil {
			return stackErr.Error(err)
		}
	}

	return nil
}

func (h *messageHandler) handleLedgerEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("handleLedgerEvent")
	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal account outbox event failed: %v", err))
	}
	log.Infow("handle ledger event", zap.String("event_name", event.EventName))
	switch event.EventName {
	case sharedevents.EventLedgerAccountTransferredToAccount:
		if err := h.handleLedgerAccountTransferredToAccount(ctx, event.EventData); err != nil {
			return stackErr.Error(err)
		}
	}

	return nil
}
