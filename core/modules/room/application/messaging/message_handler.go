package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"go-socket/core/modules/room/application/service"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/config"
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
	svc         service.Service
}

type outboxMessage struct {
	ID            int64           `json:"id"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Version       int64           `json:"version"`
	EventName     string          `json:"event_name"`
	MetaData      json.RawMessage `json:"metadata"`
	EventData     json.RawMessage `json:"event_data"`
	CreatedAt     string          `json:"created_at"`
}

func (m *outboxMessage) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return stackErr.Error(err)
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

	type alias outboxMessage
	var aux alias
	normalizedData, err := json.Marshal(normalized)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := json.Unmarshal(normalizedData, &aux); err != nil {
		return stackErr.Error(err)
	}

	*m = outboxMessage(aux)
	return nil
}

func NewMessageHandler(cfg *config.Config, repos repos.Repos, svc service.Service) (MessageHandler, error) {
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
		consumer.Read(h.processMessage(consumer))
	}
	return nil
}

func (h *messageHandler) Stop() error {
	infraMessaging.StopConsumers(h.consumer)
	return nil
}

func (h *messageHandler) handleAccountEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("handleAccountEvent")
	var event outboxMessage
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
	var event outboxMessage
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
