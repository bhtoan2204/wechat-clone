package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"wechat-clone/core/modules/room/application/service"
	"wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/contracts"
	sharedevents "wechat-clone/core/shared/contracts/events"
	infraMessaging "wechat-clone/core/shared/infra/messaging"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

//go:generate mockgen -package=messaging -destination=message_handler_mock.go -source=message_handler.go
type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer    []infraMessaging.Consumer
	accountRepo repos.RoomAccountRepository
	baseRepo    repos.Repos
	svc         service.RealtimeService
}

func NewMessageHandler(cfg *config.Config, repos repos.Repos, svc service.RealtimeService) (MessageHandler, error) {
	instance := &messageHandler{
		consumer:    make([]infraMessaging.Consumer, 0),
		accountRepo: repos.RoomAccountRepository(),
		baseRepo:    repos,
		svc:         svc,
	}

	topicHandlers := map[string]infraMessaging.Handler{}
	if topic := strings.TrimSpace(cfg.KafkaConfig.KafkaRoomConsumer.AccountTopic); topic != "" {
		topicHandlers[topic] = func(ctx context.Context, value []byte) error {
			return instance.handleAccountEvent(ctx, value)
		}
	}
	if topic := strings.TrimSpace(cfg.KafkaConfig.KafkaRoomConsumer.LedgerOutboxTopic); topic != "" {
		topicHandlers[topic] = func(ctx context.Context, value []byte) error {
			return instance.handleLedgerEvent(ctx, value)
		}
	}

	for topic, handler := range topicHandlers {
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
		consumer.SetHandler(handler)
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
		return stackErr.Error(fmt.Errorf("unmarshal account outbox event failed: %w", err))
	}
	log.Infow("handle account event", zap.String("event_name", event.EventName))
	switch event.EventName {
	case sharedevents.EventAccountCreated:
		if err := h.handleAccountCreatedEvent(ctx, event.EventData); err != nil {
			log.Errorw("handle account created event failed", zap.Error(err))
			return stackErr.Error(err)
		}
	case sharedevents.EventAccountProfileUpdated:
		if err := h.handleAccountUpdatedEvent(ctx, event.EventData); err != nil {
			log.Errorw("handle account updated event failed", zap.Error(err))
			return stackErr.Error(err)
		}
	}

	return nil
}

func (h *messageHandler) handleLedgerEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("handleLedgerEvent")
	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal ledger outbox event failed: %w", err))
	}
	log.Infow("handle ledger event", zap.String("event_name", event.EventName))
	switch event.EventName {
	case sharedevents.EventLedgerAccountTransferredToAccount:
		ctx = context.WithValue(ctx, ledgerTransferSenderAccountIDKey{}, strings.TrimSpace(event.AggregateID))
		if err := h.handleLedgerAccountTransferredToAccount(ctx, event.EventData); err != nil {
			log.Errorw("handle ledger account transferred event failed", zap.Error(err))
			return stackErr.Error(err)
		}
	}

	return nil
}
