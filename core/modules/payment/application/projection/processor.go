package projection

import (
	"context"
	"fmt"
	"strings"

	appCtx "go-socket/core/context"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/repository"
	"go-socket/core/shared/config"
	infraMessaging "go-socket/core/shared/infra/messaging"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
)

type Processor interface {
	Start() error
	Stop() error
}

type processor struct {
	consumer              []infraMessaging.Consumer
	accountProjectionRepo paymentrepos.PaymentAccountProjectionRepository
}

func NewProcessor(cfg *config.Config, appCtx *appCtx.AppContext) (Processor, error) {
	repos := repository.NewRepoImpl(appCtx)

	instance := &processor{
		consumer:              make([]infraMessaging.Consumer, 0),
		accountProjectionRepo: repos.PaymentAccountProjectionRepository(),
	}

	consumeTopics := []string{cfg.KafkaConfig.KafkaPaymentConsumer.AccountTopic}
	mapHandler := map[string]infraMessaging.Handler{
		fmt.Sprintf("payment-%s-handler", strings.ToLower(cfg.KafkaConfig.KafkaPaymentConsumer.AccountTopic)): func(ctx context.Context, value []byte) error {
			return instance.handleAccountEvent(ctx, value)
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
			return nil, stackerr.Error(err)
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
