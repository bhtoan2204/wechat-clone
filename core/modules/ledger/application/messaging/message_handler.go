package messaging

import (
	"context"
	"fmt"
	"strings"

	ledgerprojection "go-socket/core/modules/ledger/application/projection"
	"go-socket/core/modules/ledger/application/service"
	"go-socket/core/shared/config"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/contxt"
	"go-socket/core/shared/pkg/logging"
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
}

func NewMessageHandler(
	cfg *config.Config,
	ledgerService service.LedgerService,
	projector ledgerprojection.Projector,
) (MessageHandler, error) {
	instance := &messageHandler{
		consumer:      make([]infraMessaging.Consumer, 0, 1),
		ledgerService: ledgerService,
		projector:     projector,
	}

	topic := strings.TrimSpace(cfg.KafkaConfig.KafkaLedgerConsumer.PaymentOutboxTopic)
	if topic == "" {
		return instance, nil
	}

	handlerName := fmt.Sprintf("ledger-%s-handler", strings.ToLower(topic))
	consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
		Servers:      cfg.KafkaConfig.KafkaServers,
		Group:        cfg.KafkaConfig.KafkaLedgerConsumer.LedgerGroup,
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

func (h *messageHandler) processMessage(consume infraMessaging.Consumer) infraMessaging.CallBack {
	return func(ctx context.Context, _ string, vals []byte) (err error) {
		ctx = contxt.SetRequestID(ctx)

		logger := logging.FromContext(ctx)
		if reqID := contxt.RequestIDFromCtx(ctx); reqID != "" {
			logger = logger.With("request_id", reqID)
		}
		ctx = logging.WithLogger(ctx, logger)

		defer func() {
			if r := recover(); r != nil {
				err = stackErr.Error(fmt.Errorf("panic recovered: %v", r))
			}
		}()

		handler := consume.GetHandler()
		if handler == nil {
			return stackErr.Error(fmt.Errorf("consumer handler is nil"))
		}

		if err = handler(ctx, vals); err != nil {
			return stackErr.Error(err)
		}

		return nil
	}
}
