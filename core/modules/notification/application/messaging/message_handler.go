package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"go-socket/core/modules/notification/application/adapter"
	"go-socket/core/modules/notification/domain/repos"
	"go-socket/core/shared/config"
	"go-socket/core/shared/contracts"
	sharedevents "go-socket/core/shared/contracts/events"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"strings"

	"go.uber.org/zap"
)

type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer    []infraMessaging.Consumer
	emailSender adapter.EmailSender

	notificationRepo repos.NotificationRepository
}

func NewMessageHandler(cfg *config.Config, emailSender adapter.EmailSender, notificationRepo repos.NotificationRepository) (MessageHandler, error) {
	instance := &messageHandler{
		consumer:         make([]infraMessaging.Consumer, 0),
		emailSender:      emailSender,
		notificationRepo: notificationRepo,
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
		consumer.Read(h.processMessage(consumer))
	}
	return nil
}

func (h *messageHandler) Stop() error {
	for _, consumer := range h.consumer {
		consumer.Stop()
	}
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
	default:
		return nil
	}

	return nil
}
