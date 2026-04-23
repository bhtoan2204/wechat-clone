package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	notificationservice "wechat-clone/core/modules/notification/application/service"
	"wechat-clone/core/modules/notification/domain/repos"
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
	consumer []infraMessaging.Consumer
	baseRepo repos.Repos
	realtime notificationservice.RealtimeService
	push     notificationservice.PushDeliveryService
	email    notificationservice.EmailVerificationService
}

func NewMessageHandler(
	cfg *config.Config,
	baseRepo repos.Repos,
	services notificationservice.Services,
) (MessageHandler, error) {
	instance := &messageHandler{
		consumer: make([]infraMessaging.Consumer, 0),
		baseRepo: baseRepo,
		email:    services.EmailVerificationService(),
		realtime: services.RealtimeService(),
		push:     services.PushDeliveryService(),
	}

	topicHandlers := map[string]infraMessaging.Handler{}
	if topic := strings.TrimSpace(cfg.KafkaConfig.KafkaNotificationConsumer.AccountTopic); topic != "" {
		topicHandlers[topic] = func(ctx context.Context, value []byte) error {
			return instance.handleAccountEvent(ctx, value)
		}
	}
	if topic := strings.TrimSpace(cfg.KafkaConfig.KafkaNotificationConsumer.RoomOutboxTopic); topic != "" {
		topicHandlers[topic] = func(ctx context.Context, value []byte) error {
			return instance.handleRoomOutboxEvent(ctx, value)
		}
	}
	if topic := strings.TrimSpace(cfg.KafkaConfig.KafkaRelationshipConsumer.RelationshipOutboxTopic); topic != "" {
		topicHandlers[topic] = func(ctx context.Context, value []byte) error {
			return instance.handleRelationshipOutboxEvent(ctx, value)
		}
	}

	for topic, handler := range topicHandlers {
		consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
			Servers:      cfg.KafkaConfig.KafkaServers,
			Group:        cfg.KafkaConfig.KafkaNotificationConsumer.NotificationGroup,
			OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
			ConsumeTopic: []string{topic},
			HandlerName:  fmt.Sprintf("notification-%s-handler", strings.ToLower(topic)),
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
		consumer.Read(infraMessaging.WrapConsumerCallback(consumer, "Handle notification message failed"))
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
			return stackErr.Error(err)
		}
	default:
		return nil
	}

	return nil
}
