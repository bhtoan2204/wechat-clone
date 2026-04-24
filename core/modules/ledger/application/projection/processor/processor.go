package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"wechat-clone/core/modules/ledger/application/projection"
	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/contracts"
	infraMessaging "wechat-clone/core/shared/infra/messaging"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

//go:generate mockgen -package=processor -destination=processor_mock.go -source=processor.go
type Processor interface {
	Start() error
	Stop() error
}

type processor struct {
	consumer  []infraMessaging.Consumer
	projector projection.LedgerProjection
}

func NewProcessor(cfg *config.Config, ledgerProjector projection.LedgerProjection) (Processor, error) {
	instance := &processor{
		consumer:  make([]infraMessaging.Consumer, 0, 1),
		projector: ledgerProjector,
	}

	topic := strings.TrimSpace(cfg.KafkaConfig.KafkaLedgerConsumer.LedgerOutboxTopic)
	if topic == "" || ledgerProjector == nil {
		return instance, nil
	}

	handlerName := fmt.Sprintf("ledger-projection-%s-handler", strings.ToLower(topic))
	consumer, err := infraMessaging.NewConsumer(&infraMessaging.Config{
		Servers:      cfg.KafkaConfig.KafkaServers,
		Group:        cfg.KafkaConfig.KafkaLedgerConsumer.LedgerProjectionGroup,
		OffsetReset:  cfg.KafkaConfig.KafkaOffsetReset,
		ConsumeTopic: []string{topic},
		HandlerName:  handlerName,
		DLQ:          true,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	consumer.SetHandler(func(ctx context.Context, value []byte) error {
		return instance.handleLedgerOutboxEvent(ctx, value)
	})
	instance.consumer = append(instance.consumer, consumer)

	return instance, nil
}

func (p *processor) Start() error {
	for _, consumer := range p.consumer {
		consumer.Read(infraMessaging.WrapConsumerCallback(consumer, "Handle ledger projection message failed"))
	}
	return nil
}

func (p *processor) Stop() error {
	infraMessaging.StopConsumers(p.consumer)
	return nil
}

func (h *processor) handleLedgerOutboxEvent(ctx context.Context, value []byte) error {
	if h.projector == nil {
		return nil
	}

	log := logging.FromContext(ctx).Named("LedgerProjectionEvent")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal ledger outbox event failed: %w", err))
	}

	log.Infow("handle ledger outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	if !projection.IsLedgerTransactionProjectionEvent(event.EventName) {
		return nil
	}

	payload, err := toLedgerTransactionProjection(event)
	if err != nil {
		return stackErr.Error(err)
	}
	return stackErr.Error(h.projector.ProjectTransaction(ctx, &payload))
}

func toLedgerTransactionProjection(event contracts.OutboxMessage) (projection.LedgerTransactionProjected, error) {
	outboxCreatedAt, err := parseOutboxCreatedAt(event.CreatedAt)
	if err != nil {
		return projection.LedgerTransactionProjected{}, stackErr.Error(err)
	}

	payload, err := unmarshalLedgerAggregateEvent(event.EventName, event.EventData)
	if err != nil {
		return projection.LedgerTransactionProjected{}, stackErr.Error(err)
	}

	posting, ok, err := ledgeraggregate.NewLedgerAccountPostingFromEvent(strings.TrimSpace(event.AggregateID), payload)
	if err != nil {
		return projection.LedgerTransactionProjected{}, stackErr.Error(err)
	}
	if !ok {
		return projection.LedgerTransactionProjected{}, stackErr.Error(fmt.Errorf(
			"unsupported ledger projection event: aggregate_id=%s event_name=%s",
			event.AggregateID,
			event.EventName,
		))
	}
	if posting.BookedAt.IsZero() {
		posting.BookedAt = outboxCreatedAt
	}

	return projection.LedgerTransactionProjected{
		TransactionID: posting.TransactionID,
		ReferenceType: posting.ReferenceType,
		ReferenceID:   posting.ReferenceID,
		Currency:      posting.Currency,
		CreatedAt:     posting.BookedAt.UTC(),
		Entries: []projection.LedgerTransactionEntry{
			{
				AccountID: strings.TrimSpace(event.AggregateID),
				Currency:  posting.Currency,
				Amount:    posting.AmountDelta,
				CreatedAt: posting.BookedAt.UTC(),
			},
		},
	}, nil
}

func unmarshalLedgerAggregateEvent(eventName string, data json.RawMessage) (interface{}, error) {
	switch strings.TrimSpace(eventName) {
	case ledgeraggregate.EventNameLedgerAccountDepositFromIntent:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountDepositFromIntent](data, "unmarshal ledger deposit from intent payload failed")
	case ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountWithdrawFromIntent](data, "unmarshal ledger withdraw from intent payload failed")
	case ledgeraggregate.EventNameLedgerAccountDepositFromRefund:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountDepositFromRefund](data, "unmarshal ledger deposit from refund payload failed")
	case ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountWithdrawFromRefund](data, "unmarshal ledger withdraw from refund payload failed")
	case ledgeraggregate.EventNameLedgerAccountDepositFromChargeback:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountDepositFromChargeback](data, "unmarshal ledger deposit from chargeback payload failed")
	case ledgeraggregate.EventNameLedgerAccountWithdrawFromChargeback:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountWithdrawFromChargeback](data, "unmarshal ledger withdraw from chargeback payload failed")
	case ledgeraggregate.EventNameLedgerAccountReserveWithdrawal:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountReserveWithdrawal](data, "unmarshal ledger reserve withdrawal payload failed")
	case ledgeraggregate.EventNameLedgerAccountReceiveWithdrawalHold:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountReceiveWithdrawalHold](data, "unmarshal ledger receive withdrawal hold payload failed")
	case ledgeraggregate.EventNameLedgerAccountReleaseWithdrawal:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountReleaseWithdrawal](data, "unmarshal ledger release withdrawal payload failed")
	case ledgeraggregate.EventNameLedgerAccountWithdrawReleasedHold:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountWithdrawReleasedHold](data, "unmarshal ledger withdraw released hold payload failed")
	case ledgeraggregate.EventNameLedgerAccountTransferredToAccount:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountTransferredToAccount](data, "unmarshal ledger transfer out payload failed")
	case ledgeraggregate.EventNameLedgerAccountReceivedTransfer:
		return unmarshalLedgerEventData[ledgeraggregate.EventLedgerAccountReceivedTransfer](data, "unmarshal ledger transfer in payload failed")
	default:
		return nil, stackErr.Error(fmt.Errorf("unsupported ledger event_name=%s", eventName))
	}
}

func unmarshalLedgerEventData[T any](data json.RawMessage, message string) (*T, error) {
	var payload T
	if err := contracts.UnmarshalEventData(data, &payload); err != nil {
		return nil, stackErr.Error(fmt.Errorf("%s: %w", message, err))
	}
	return &payload, nil
}

func parseOutboxCreatedAt(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, stackErr.Error(fmt.Errorf("ledger outbox created_at is required"))
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, stackErr.Error(fmt.Errorf("parse ledger outbox created_at failed: %w", err))
	}
	return parsed.UTC(), nil
}
