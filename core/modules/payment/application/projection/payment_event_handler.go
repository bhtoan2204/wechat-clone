package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-socket/core/modules/payment/domain/aggregate"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/domain/types"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (p *processor) handlePaymentEvent(ctx context.Context, value []byte) error {
	log := logging.FromContext(ctx).Named("PaymentProjection")
	var event paymentEventMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackerr.Error(fmt.Errorf("unmarshal payment event failed: %w", err))
	}

	log.Infow("handle payment event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
		zap.Int64("version", event.Version),
	)

	switch event.EventName {
	case "EventPaymentTransactionDeposited":
		return p.projectDepositedEvent(ctx, &event)
	case "EventPaymentTransactionWithdrawn":
		return p.projectWithdrawnEvent(ctx, &event)
	case "EventPaymentTransactionTransferred":
		return p.projectTransferredEvent(ctx, &event)
	case "EventPaymentTransactionReceived":
		return p.projectReceivedEvent(ctx, &event)
	default:
		return nil
	}
}

func (p *processor) projectDepositedEvent(ctx context.Context, event *paymentEventMessage) error {
	payloadAny, err := decodeEventPayload(ctx, p.eventSerializer, event.AggregateType, event.EventName, event.EventData)
	if err != nil {
		return stackerr.Error(fmt.Errorf("decode deposit payload failed: %w", err))
	}

	payload, ok := payloadAny.(*aggregate.EventPaymentTransactionDeposited)
	if !ok || payload == nil {
		return stackerr.Error(fmt.Errorf("invalid deposit payload"))
	}

	return p.projectTransaction(ctx, event.ID, payload.PaymentTransactionID, event.AggregateID, payload.PaymentTransactionAmount, payload.PaymentTransactionAmount, types.TransactionTypeDeposited, payload.PaymentTransactionCreatedAt)
}

func (p *processor) projectWithdrawnEvent(ctx context.Context, event *paymentEventMessage) error {
	payloadAny, err := decodeEventPayload(ctx, p.eventSerializer, event.AggregateType, event.EventName, event.EventData)
	if err != nil {
		return stackerr.Error(fmt.Errorf("decode withdrawal payload failed: %w", err))
	}

	payload, ok := payloadAny.(*aggregate.EventPaymentTransactionWithdrawn)
	if !ok || payload == nil {
		return stackerr.Error(fmt.Errorf("invalid withdrawal payload"))
	}

	return p.projectTransaction(ctx, event.ID, payload.PaymentTransactionID, event.AggregateID, payload.PaymentTransactionAmount, -payload.PaymentTransactionAmount, types.TransactionTypeWithdrawn, payload.PaymentTransactionCreatedAt)
}

func (p *processor) projectTransferredEvent(ctx context.Context, event *paymentEventMessage) error {
	payloadAny, err := decodeEventPayload(ctx, p.eventSerializer, event.AggregateType, event.EventName, event.EventData)
	if err != nil {
		return stackerr.Error(fmt.Errorf("decode transfer payload failed: %w", err))
	}

	payload, ok := payloadAny.(*aggregate.EventPaymentTransactionTransferred)
	if !ok || payload == nil {
		return stackerr.Error(fmt.Errorf("invalid transfer payload"))
	}

	return p.projectTransaction(ctx, event.ID, payload.PaymentTransactionID, event.AggregateID, payload.PaymentTransactionAmount, -payload.PaymentTransactionAmount, types.TransactionTypeTransferred, payload.PaymentTransactionCreatedAt)
}

func (p *processor) projectReceivedEvent(ctx context.Context, event *paymentEventMessage) error {
	payloadAny, err := decodeEventPayload(ctx, p.eventSerializer, event.AggregateType, event.EventName, event.EventData)
	if err != nil {
		return stackerr.Error(fmt.Errorf("decode receive payload failed: %w", err))
	}

	payload, ok := payloadAny.(*aggregate.EventPaymentTransactionReceived)
	if !ok || payload == nil {
		return stackerr.Error(fmt.Errorf("invalid receive payload"))
	}

	return p.projectTransaction(ctx, event.ID, payload.PaymentTransactionID, event.AggregateID, payload.PaymentTransactionAmount, payload.PaymentTransactionAmount, types.TransactionTypeReceived, payload.PaymentTransactionCreatedAt)
}

func (p *processor) projectTransaction(ctx context.Context, eventID, transactionID, accountID string, amount, balanceDelta int64, transactionType types.TransactionType, createdAt time.Time) error {
	return p.repos.WithTransaction(ctx, func(txRepos paymentrepos.Repos) error {
		if err := txRepos.PaymentProjectionRepository().ProjectTransaction(ctx, eventID, transactionID, accountID, amount, balanceDelta, transactionType, createdAt); err != nil {
			return stackerr.Error(err)
		}
		return nil
	})
}
