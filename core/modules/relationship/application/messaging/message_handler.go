package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/contracts"
	sharedevents "wechat-clone/core/shared/contracts/events"
	infraMessaging "wechat-clone/core/shared/infra/messaging"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer    []infraMessaging.Consumer
	accountRepo repos.RelationshipAccountRepository
}

func NewMessageHandler(cfg *config.Config, baseRepo repos.Repos) (MessageHandler, error) {
	instance := &messageHandler{
		consumer:    make([]infraMessaging.Consumer, 0, 1),
		accountRepo: baseRepo.RelationshipAccountRepository(),
	}

	accountTopic := strings.TrimSpace(cfg.KafkaConfig.KafkaRelationshipConsumer.AccountTopic)
	if accountTopic == "" {
		return instance, nil
	}

	consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
		Servers:      cfg.KafkaConfig.KafkaServers,
		Group:        cfg.KafkaConfig.KafkaRelationshipConsumer.RelationshipProjectionGroup,
		OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
		ConsumeTopic: []string{accountTopic},
		HandlerName:  fmt.Sprintf("relationship-%s-handler", strings.ToLower(accountTopic)),
		DLQ:          true,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	consumer.SetHandler(func(ctx context.Context, value []byte) error {
		return instance.handleAccountEvent(ctx, value)
	})
	instance.consumer = append(instance.consumer, consumer)

	return instance, nil
}

func (h *messageHandler) Start() error {
	for _, consumer := range h.consumer {
		consumer.Read(infraMessaging.WrapConsumerCallback(consumer, "Handle relationship message failed"))
	}
	return nil
}

func (h *messageHandler) Stop() error {
	infraMessaging.StopConsumers(h.consumer)
	return nil
}

func (h *messageHandler) handleAccountEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("RelationshipHandleAccountEvent")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal account outbox event failed: %w", err))
	}

	log.Infow("handle relationship account event", zap.String("event_name", event.EventName))

	switch event.EventName {
	case sharedevents.EventAccountCreated:
		return stackErr.Error(h.handleAccountCreatedEvent(ctx, event.EventData))
	case sharedevents.EventAccountProfileUpdated:
		return stackErr.Error(h.handleAccountUpdatedEvent(ctx, event.EventData))
	default:
		return nil
	}
}
