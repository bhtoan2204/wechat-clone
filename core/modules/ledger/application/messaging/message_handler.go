package messaging

import (
	"context"
	"fmt"
	"strings"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/ledger/application/service"
	ledgerrepo "wechat-clone/core/modules/ledger/infra/persistent/repository"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/contracts"
	"wechat-clone/core/shared/infra/lock"
	infraMessaging "wechat-clone/core/shared/infra/messaging"
	"wechat-clone/core/shared/pkg/stackErr"
)

type paymentOutboxMessage = contracts.OutboxMessage

//go:generate mockgen -package=messaging -destination=message_handler_mock.go -source=message_handler.go
type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer      []infraMessaging.Consumer
	ledgerService service.LedgerService
	locker        lock.Lock
}

func NewMessageHandler(
	cfg *config.Config,
	appCtx *appCtx.AppContext,
) (MessageHandler, error) {
	ledgerSvc := service.NewLedgerService(ledgerrepo.NewRepoImpl(appCtx))

	instance := &messageHandler{
		consumer:      make([]infraMessaging.Consumer, 0, 1),
		ledgerService: ledgerSvc,
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
