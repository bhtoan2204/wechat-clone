package projection

import (
	"context"
	"fmt"
	"strings"

	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/config"
	infraMessaging "go-socket/core/shared/infra/messaging"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
)

//go:generate mockgen -package=projection -destination=processor_mock.go -source=processor.go
type Processor interface {
	Start() error
	Stop() error
}

type processor struct {
	consumer              []infraMessaging.Consumer
	repos                 paymentrepos.Repos
	accountProjectionRepo paymentrepos.PaymentAccountProjectionRepository
	eventSerializer       eventpkg.Serializer
}

func NewProcessor(cfg *config.Config, repos paymentrepos.Repos) (Processor, error) {
	eventSerializer, err := newProjectionSerializer()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	instance := &processor{
		consumer:              make([]infraMessaging.Consumer, 0),
		repos:                 repos,
		accountProjectionRepo: repos.PaymentAccountProjectionRepository(),
		eventSerializer:       eventSerializer,
	}

	consumeTopics := []string{
		cfg.KafkaConfig.KafkaPaymentConsumer.AccountTopic,
		cfg.KafkaConfig.KafkaPaymentConsumer.PaymentEventsTopic,
	}
	mapHandler := map[string]infraMessaging.Handler{
		fmt.Sprintf("payment-%s-handler", strings.ToLower(cfg.KafkaConfig.KafkaPaymentConsumer.AccountTopic)): func(ctx context.Context, value []byte) error {
			return instance.handleAccountEvent(ctx, value)
		},
		fmt.Sprintf("payment-%s-handler", strings.ToLower(cfg.KafkaConfig.KafkaPaymentConsumer.PaymentEventsTopic)): func(ctx context.Context, value []byte) error {
			return instance.handlePaymentEvent(ctx, value)
		},
	}

	for _, topic := range consumeTopics {
		consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
			Servers:      cfg.KafkaConfig.KafkaServers,
			Group:        cfg.KafkaConfig.KafkaPaymentConsumer.PaymentGroup,
			OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
			ConsumeTopic: []string{topic},
			HandlerName:  fmt.Sprintf("payment-%s-handler", strings.ToLower(topic)),
			DLQ:          true,
		})
		if err != nil {
			return nil, stackErr.Error(err)
		}
		consumer.SetHandler(mapHandler[fmt.Sprintf("payment-%s-handler", strings.ToLower(topic))])
		instance.consumer = append(instance.consumer, consumer)
	}

	return instance, nil
}

func (p *processor) Start() error {
	log := logging.DefaultLogger().Named("PaymentProcessor")
	log.Info("Starting payment processor")
	for _, consumer := range p.consumer {
		consumer.Read(p.processMessage(consumer))
	}
	return nil
}

func (p *processor) Stop() error {
	for _, consumer := range p.consumer {
		consumer.Stop()
	}
	return nil
}
