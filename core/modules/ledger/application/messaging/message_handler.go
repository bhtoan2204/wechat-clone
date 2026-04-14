package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ledgerservice "go-socket/core/modules/ledger/application/service"
	"go-socket/core/shared/config"
	sharedevents "go-socket/core/shared/contracts/events"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/contxt"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

//go:generate mockgen -package=messaging -destination=message_handler_mock.go -source=message_handler.go
type MessageHandler interface {
	Start() error
	Stop() error
}

type messageHandler struct {
	consumer      []infraMessaging.Consumer
	ledgerService *ledgerservice.LedgerService
}

func NewMessageHandler(cfg *config.Config, ledgerService *ledgerservice.LedgerService) (MessageHandler, error) {
	instance := &messageHandler{
		consumer:      make([]infraMessaging.Consumer, 0, 1),
		ledgerService: ledgerService,
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

func (h *messageHandler) handlePaymentOutboxEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("LedgerPaymentEvent")

	var event paymentOutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal payment outbox event failed: %v", err))
	}

	log.Infow("handle payment outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	switch event.EventName {
	case sharedevents.EventPaymentSucceeded:
		var payload sharedevents.PaymentSucceededEvent
		if err := json.Unmarshal(event.EventData, &payload); err != nil {
			return stackErr.Error(fmt.Errorf("unmarshal payment succeeded payload failed: %v", err))
		}
		if payload.PaymentID == "" {
			payload.PaymentID = event.AggregateID
		}
		return h.ledgerService.RecordPaymentSucceeded(ctx, &payload)
	default:
		return nil
	}
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
