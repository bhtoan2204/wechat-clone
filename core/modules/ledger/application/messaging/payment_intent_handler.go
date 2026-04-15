package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	ledgerservice "go-socket/core/modules/ledger/application/service"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

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
			var raw string
			if err2 := json.Unmarshal(event.EventData, &raw); err2 != nil {
				return stackErr.Error(fmt.Errorf("unmarshal payment succeeded payload failed: %v", err))
			}
			if err2 := json.Unmarshal([]byte(raw), &payload); err2 != nil {
				return stackErr.Error(fmt.Errorf("unmarshal inner payload failed: %v", err2))
			}
		}
		if payload.PaymentID == "" {
			payload.PaymentID = event.AggregateID
		}

		return h.ledgerService.RecordPaymentSucceeded(ctx, ledgerservice.RecordPaymentSucceededCommand{
			PaymentID:          payload.PaymentID,
			TransactionID:      payload.TransactionID,
			ClearingAccountKey: payload.ClearingAccountKey,
			CreditAccountID:    payload.CreditAccountID,
			Currency:           payload.Currency,
			Amount:             payload.Amount,
		})
	default:
		return nil
	}
}
