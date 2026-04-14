package messaging

import (
	"context"
	"fmt"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type CallBack func(ctx context.Context, topic string, value []byte) error
type Handler func(ctx context.Context, value []byte) error

//go:generate mockgen -package=messaging -destination=consumer_mock.go -source=consumer.go
type Consumer interface {
	Read(callback CallBack)
	Stop()
	SetHandler(h Handler)
	GetHandler() Handler
	GetHandlerName() string
}

//go:generate mockgen -package=messaging -destination=consumer_mock.go -source=consumer.go
type kafkaConsumerClient interface {
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error
	Poll(timeoutMs int) kafka.Event
	Assign(partitions []kafka.TopicPartition) error
	Unassign() error
	Unsubscribe() error
	Close() error
	CommitMessage(msg *kafka.Message) ([]kafka.TopicPartition, error)
	Seek(partition kafka.TopicPartition, timeoutMs int) error
}

type consumer struct {
	instance kafkaConsumerClient

	startSingleton sync.Once
	stopSingleton  sync.Once

	chanStop    chan bool
	handler     Handler
	handlerName string

	dlq          bool
	retryBackoff time.Duration
}

func NewConsumer(cfg *Config) (Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, stackErr.Error(err)
	}
	cfgMap := &kafka.ConfigMap{
		"bootstrap.servers":        cfg.Servers,
		"group.id":                 cfg.Group,
		"auto.offset.reset":        cfg.OffsetReset,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,
		"session.timeout.ms":       120000,
		"heartbeat.interval.ms":    3000,
		"max.poll.interval.ms":     600000,
	}
	c, err := kafka.NewConsumer(cfgMap)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := c.SubscribeTopics(cfg.ConsumeTopic, nil); err != nil {
		return nil, stackErr.Error(err)
	}
	return &consumer{
		instance:     c,
		chanStop:     make(chan bool, 1),
		handlerName:  cfg.HandlerName,
		dlq:          cfg.DLQ,
		retryBackoff: time.Second,
	}, nil
}

func (c *consumer) Read(f CallBack) {
	c.startSingleton.Do(func() {
		go func() {
			c.start(f)
		}()
	})
}

func (c *consumer) SetHandler(f Handler) {
	if c.handler == nil {
		c.handler = f
	}
}

func (c *consumer) GetHandler() Handler {
	return c.handler
}

func (c *consumer) GetHandlerName() string {
	return c.handlerName
}

func (c *consumer) start(f CallBack) {
	log := logging.DefaultLogger()

loop:
	for {
		select {
		case <-c.chanStop:
			log.Infow("Caught signal stop kafa consumer, terminating ...")
			break loop
		default:
			ev := c.instance.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				c.handleMessage(log, f, e)
			case kafka.Error:
				if !e.IsTimeout() {
					log.Errorw("Consume kafka got error", zap.Error(e))
				}
			case kafka.AssignedPartitions:
				log.Warnw("Partitions assigned", zap.Any("partitions", e.Partitions))
				err := c.instance.Assign(e.Partitions)
				if err != nil {
					log.Errorw("Failed to assign partitions", zap.Error(err))
				}
			case kafka.RevokedPartitions:
				log.Warnw("Partitions revoked", zap.Any("partitions", e.Partitions))
				err := c.instance.Unassign()
				if err != nil {
					log.Errorw("Failed to unassign partitions", zap.Error(err))
				}
			default:
			}
		}
	}
}

func (c *consumer) Stop() {
	c.stopSingleton.Do(func() {
		log := logging.DefaultLogger()
		log.Infow("Stopping Kafka consumer gracefully...")

		c.chanStop <- true

		time.Sleep(500 * time.Millisecond)

		if err := c.instance.Unsubscribe(); err != nil {
			log.Warnw("Failed to unsubscribe", zap.Error(err))
		}

		if err := c.instance.Close(); err != nil {
			log.Warnw("Failed to close consumer", zap.Error(err))
		}

		log.Infow("Kafka consumer stopped successfully")
	})
}

func (c *consumer) handleMessage(log *zap.SugaredLogger, f CallBack, msg *kafka.Message) {
	var topic string
	if msg.TopicPartition.Topic != nil {
		topic = *msg.TopicPartition.Topic
	}

	ctx, span := c.startSpan(msg)
	defer span.End()

	err := processMessageWithRetry(ctx, f, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		log.Errorw("consumer process got error",
			zap.String("topic", topic), zap.Error(err),
			zap.ByteString("val", msg.Value),
			zap.ByteString("key", msg.Key),
			zap.Int64("offset", int64(msg.TopicPartition.Offset)))

		if c.dlq {
			c.StoreDLQ(ctx, msg)
		}

		// Rewind to the failed offset so later commits cannot skip an unprocessed message.
		if rewindErr := c.rewindMessage(msg); rewindErr != nil {
			span.RecordError(rewindErr)
			log.Warnw("Failed to rewind offset after message processing error",
				zap.Any("topic_partition", msg.TopicPartition),
				zap.Error(rewindErr))
		}

		if c.retryBackoff > 0 {
			time.Sleep(c.retryBackoff)
		}
		return
	}

	if _, err := c.instance.CommitMessage(msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Warnw("Failed to commit offset after message processing",
			zap.Any("topic_partition", msg.TopicPartition),
			zap.Error(err))
	}
}

func (c *consumer) rewindMessage(msg *kafka.Message) error {
	return c.instance.Seek(msg.TopicPartition, int((5*time.Second)/time.Millisecond))
}

const DLQSuffix = "dlq"

func (c *consumer) StoreDLQ(ctx context.Context, msg *kafka.Message) {
	// topic := *msg.TopicPartition.Topic
	// c.producer.ProduceRawWithKey(ctx, GetDLQTopic(topic), msg.Key, msg.Value)
}

func GetDLQTopic(topic string) string {
	if !strings.HasSuffix(topic, DLQSuffix) {
		topic = fmt.Sprintf("%s.%s", topic, DLQSuffix)
	}

	return topic
}

func processMessageWithRetry(ctx context.Context, f CallBack, msg *kafka.Message) error {
	retryTimes := uint(3)
	topic := topicFromMessage(msg)
	if strings.HasSuffix(topic, DLQSuffix) {
		retryTimes = 0
	}

	options := []retry.Option{
		retry.Attempts(retryTimes),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.MaxDelay(time.Second * 5),
	}

	return retry.Do(func() error {
		return f(ctx, topic, msg.Value)
	}, options...)
}
