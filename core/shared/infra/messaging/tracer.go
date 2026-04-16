package messaging

import (
	"context"
	"fmt"
	"go-socket/core/shared/infra/xtracer"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func (c *consumer) startSpan(msg *kafka.Message) (context.Context, trace.Span) {
	carrier := NewMessageCarrier(msg)
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	topic := topicFromMessage(msg)

	namespan := fmt.Sprintf("%s receive", topic)
	opts := c.buildSpanOpts(msg)

	return xtracer.StartSpan(ctx, namespan, opts...)
}

func (c *consumer) buildSpanOpts(msg *kafka.Message) []trace.SpanStartOption {
	result := []trace.SpanStartOption{}
	offset := strconv.FormatInt(int64(msg.TopicPartition.Offset), 10)
	topic := topicFromMessage(msg)

	result = append(result,
		trace.WithAttributes(
			semconv.MessagingSourceNameKey.String(topic),
			semconv.MessagingMessageIDKey.String(offset),
			semconv.MessagingKafkaMessageKeyKey.String(string(msg.Key)),
			semconv.MessagingKafkaSourcePartitionKey.Int64(int64(msg.TopicPartition.Partition)),
		),
		trace.WithSpanKind(trace.SpanKindConsumer),
	)

	return result
}

func topicFromMessage(msg *kafka.Message) string {
	if msg.TopicPartition.Topic == nil {
		return "unknown"
	}
	return *msg.TopicPartition.Topic
}

func (p *producer) startSpan(ctx context.Context, msg *kafka.Message) (context.Context, trace.Span) {
	carrier := NewMessageCarrier(msg)
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	namespan := fmt.Sprintf("%s send", *msg.TopicPartition.Topic)
	opts := p.buildSpanOpts(msg)

	return xtracer.StartSpan(ctx, namespan, opts...)
}

func (c *producer) buildSpanOpts(msg *kafka.Message) []trace.SpanStartOption {
	result := []trace.SpanStartOption{}
	offset := strconv.FormatInt(int64(msg.TopicPartition.Offset), 10)

	result = append(result,
		trace.WithAttributes(
			semconv.MessagingDestinationNameKey.String(*msg.TopicPartition.Topic),
			semconv.MessagingMessageIDKey.String(offset),
			semconv.MessagingKafkaMessageKeyKey.String(string(msg.Key)),
			semconv.MessagingKafkaSourcePartitionKey.Int64(int64(msg.TopicPartition.Partition)),
		),
		trace.WithSpanKind(trace.SpanKindProducer),
	)

	return result
}
