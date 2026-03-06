package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"go-socket/core/modules/notification/application/adapter"
	"go-socket/core/shared/config"
	"go-socket/core/shared/contracts/events"
	infraMessaging "go-socket/core/shared/infra/messaging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"strings"
)

type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer    []infraMessaging.Consumer
	emailSender adapter.EmailSender
}

type accountOutboxMessage struct {
	ID            int64           `json:"id"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Version       int64           `json:"version"`
	EventName     string          `json:"event_name"`
	EventData     json.RawMessage `json:"event_data"`
	CreatedAt     string          `json:"created_at"`
}

func (m *accountOutboxMessage) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	normalized := make(map[string]json.RawMessage, len(raw))

	// Keep exact lowercase keys first if present.
	for key, value := range raw {
		lowerKey := strings.ToLower(key)
		if key == lowerKey {
			normalized[lowerKey] = value
		}
	}

	// Fill remaining keys by case-insensitive mapping.
	for key, value := range raw {
		lowerKey := strings.ToLower(key)
		if _, exists := normalized[lowerKey]; !exists {
			normalized[lowerKey] = value
		}
	}

	type alias accountOutboxMessage
	var aux alias
	normalizedData, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(normalizedData, &aux); err != nil {
		return err
	}

	*m = accountOutboxMessage(aux)
	return nil
}

func NewMessageHandler(cfg *config.Config, emailSender adapter.EmailSender) (MessageHandler, error) {
	if emailSender == nil {
		return nil, stackerr.Error(fmt.Errorf("email sender can not be nil"))
	}

	instance := &messageHandler{
		emailSender: emailSender,
		consumer:    make([]infraMessaging.Consumer, 0),
	}

	consumeTopics := []string{cfg.KafkaConfig.KafkaNotificationConsumer.AccountTopic}
	mapHandler := map[string]infraMessaging.Handler{
		fmt.Sprintf("notification-%s-handler", strings.ToLower(cfg.KafkaConfig.KafkaNotificationConsumer.AccountTopic)): func(ctx context.Context, value []byte) error {
			return instance.handleAccountEvent(ctx, value)
		},
	}

	for _, topic := range consumeTopics {
		consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
			Servers:      cfg.KafkaConfig.KafkaServers,
			Group:        cfg.KafkaConfig.KafkaNotificationConsumer.NotificationGroup,
			OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
			ConsumeTopic: []string{topic},
			HandlerName:  fmt.Sprintf("notification-%s-handler", strings.ToLower(topic)),
			DLQ:          true,
		})
		if err != nil {
			return nil, stackerr.Error(err)
		}
		consumer.SetHandler(mapHandler[fmt.Sprintf("notification-%s-handler", strings.ToLower(topic))])
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
	var event accountOutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackerr.Error(fmt.Errorf("unmarshal account outbox event failed: %w", err))
	}

	switch event.EventName {
	case events.AccountCreatedEventName:
		if err := h.handleAccountCreatedEvent(ctx, event.EventData); err != nil {
			return err
		}
	default:
		return nil
	}

	return nil
}
