package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	ledgerprojection "go-socket/core/modules/ledger/application/projection"
	"go-socket/core/shared/contracts"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

func (h *messageHandler) handleLedgerOutboxEvent(ctx context.Context, value []byte) error {
	if h.projector == nil {
		return nil
	}

	log := logging.FromContext(ctx).Named("LedgerProjectionEvent")

	var event contracts.OutboxMessage
	if err := json.Unmarshal(value, &event); err != nil {
		return stackErr.Error(fmt.Errorf("unmarshal ledger outbox event failed: %v", err))
	}

	log.Infow("handle ledger outbox event",
		zap.String("event_name", event.EventName),
		zap.String("aggregate_id", event.AggregateID),
	)

	if !ledgerprojection.IsLedgerTransactionProjectionEvent(event.EventName) {
		return nil
	}

	payload, err := unmarshalLedgerTransactionProjectedPayload(event.EventData)
	if err != nil {
		return stackErr.Error(err)
	}
	return stackErr.Error(h.projector.ProjectTransaction(ctx, &payload))
}

func unmarshalLedgerTransactionProjectedPayload(data json.RawMessage) (ledgerprojection.LedgerTransactionProjected, error) {
	var payload ledgerprojection.LedgerTransactionProjected
	if err := contracts.UnmarshalEventData(data, &payload); err != nil {
		return ledgerprojection.LedgerTransactionProjected{}, stackErr.Error(fmt.Errorf("unmarshal ledger transaction projected payload failed: %v", err))
	}
	return payload, nil
}
