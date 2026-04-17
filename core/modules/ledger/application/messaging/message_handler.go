package messaging

import (
	"context"
	"fmt"
	"strings"

	appCtx "go-socket/core/context"
	ledgerprojection "go-socket/core/modules/ledger/application/projection"
	"go-socket/core/modules/ledger/application/service"
	ledgerrepo "go-socket/core/modules/ledger/infra/persistent/repository"
	"go-socket/core/shared/config"
	"go-socket/core/shared/infra/lock"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/stackErr"
)

//go:generate mockgen -package=messaging -destination=message_handler_mock.go -source=message_handler.go
type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer      []infraMessaging.Consumer
	ledgerService service.LedgerService
	projector     ledgerprojection.Projector
	locker        lock.Lock
}

func NewMessageHandler(
	cfg *config.Config,
	appCtx *appCtx.AppContext,
) (MessageHandler, error) {
	ledgerSvc := service.NewLedgerService(ledgerrepo.NewRepoImpl(appCtx))
	projector := ledgerrepo.NewLedgerProjectionRepoImpl(appCtx.GetDB())

	instance := &messageHandler{
		consumer:      make([]infraMessaging.Consumer, 0, 1),
		ledgerService: ledgerSvc,
		projector:     projector,
		locker:        appCtx.Locker(),
	}

	topic := strings.TrimSpace(cfg.KafkaConfig.KafkaLedgerConsumer.PaymentOutboxTopic)
	if topic == "" {
		return instance, nil
	}

	handlerName := fmt.Sprintf("ledger-%s-handler", strings.ToLower(topic))
	consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
		Servers:      cfg.KafkaConfig.KafkaServers,
		Group:        cfg.KafkaConfig.KafkaLedgerConsumer.LedgerMessagingGroup,
		OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
		ConsumeTopic: []string{topic},
		HandlerName:  handlerName,
		DLQ:          true,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}
	consumer.SetHandler(func(ctx context.Context, value []byte) error {
		return instance.handlePaymentOutboxEvent(ctx, value)
	})
	instance.consumer = append(instance.consumer, consumer)

	ledgerOutboxTopic := strings.TrimSpace(cfg.KafkaConfig.KafkaLedgerConsumer.LedgerOutboxTopic)
	if ledgerOutboxTopic != "" && projector != nil {
		handlerName := fmt.Sprintf("ledger-%s-projection-handler", strings.ToLower(ledgerOutboxTopic))
		projectionConsumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
			Servers:      cfg.KafkaConfig.KafkaServers,
			Group:        cfg.KafkaConfig.KafkaLedgerConsumer.LedgerProjectionGroup,
			OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
			ConsumeTopic: []string{ledgerOutboxTopic},
			HandlerName:  handlerName,
			DLQ:          true,
		})
		if err != nil {
			return nil, stackErr.Error(err)
		}
		projectionConsumer.SetHandler(func(ctx context.Context, value []byte) error {
			return instance.handleLedgerOutboxEvent(ctx, value)
		})
		instance.consumer = append(instance.consumer, projectionConsumer)
	}

	return instance, nil
}

func (h *messageHandler) Start() error {
	for _, consumer := range h.consumer {
		consumer.Read(infraMessaging.WrapConsumerCallback(consumer, "Handle ledger message failed"))
	}
	return nil
}

func (h *messageHandler) Stop() error {
	infraMessaging.StopConsumers(h.consumer)
	return nil
}
