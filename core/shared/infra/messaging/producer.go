package messaging

import (
	"context"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

//go:generate mockgen -package=messaging -destination=producer_mock.go -source=producer.go
type Producer interface {
	ProduceRawWithKey(ctx context.Context, topic string, key []byte, data []byte) error
	Close(ctx context.Context)
}

type producer struct {
	instance         *kafka.Producer
	chanStop         chan bool
	startClosingOnce sync.Once
}

func NewProducer(cfg *Config) (Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Servers,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	producer := &producer{
		instance: p,
		chanStop: make(chan bool, 1),
	}

	go producer.listenDefaultEvent()

	return producer, nil
}

func (p *producer) ProduceRawWithKey(ctx context.Context, topic string, key []byte, data []byte) error {
	log := logging.FromContext(ctx)

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: data,
	}

	_, span := p.startSpan(ctx, msg)
	defer span.End()

	if err := p.instance.Produce(msg, nil); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Errorw("producer ProduceRawWithKey got error",
			zap.Error(err), zap.ByteString("key", key), zap.ByteString("body", data))
		return err
	}

	return nil
}

func (p *producer) listenDefaultEvent() {
	log := logging.DefaultLogger().Named("KafkaProducer")

loop:
	for {
		select {
		case <-p.chanStop:
			log.Infof("Stop listen the default events channel - kafka producer")
			break loop
		case e := <-p.instance.Events():
			switch ev := e.(type) {
			case *kafka.Message:
				m := ev
				if m.TopicPartition.Error != nil {
					log.Errorw("Delivery message failed",
						zap.Error(m.TopicPartition.Error),
						zap.Int("partition", int(m.TopicPartition.Partition)),
						zap.String("value", string(m.Value)),
						zap.String("key", string(m.Key)),
						zap.Any("headers", m.Headers),
						zap.Time("timestamp", m.Timestamp),
					)
				} else {
					log.Infow("Delivered message to topic", zap.String("topic", *m.TopicPartition.Topic), zap.Int("partition", int(m.TopicPartition.Partition)), zap.Int64("offset", int64(m.TopicPartition.Offset)))
				}
			default:
			}
		}
	}
}

func (p *producer) Close(ctx context.Context) {
	log := logging.DefaultLogger().Named("KafkaProducer")

	p.startClosingOnce.Do(func() {
		log.Infof("Stoping kafka producer")
		for p.instance.Flush(5000) > 0 {
			log.Infof("Still waiting to flush outstanding messages")
		}
		p.instance.Close()
		p.chanStop <- true
		log.Infof("Kafka producer stopped")
	})
}
