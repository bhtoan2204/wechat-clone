package messaging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	ledgerprojection "go-socket/core/modules/ledger/application/projection"
	ledgeraggregate "go-socket/core/modules/ledger/domain/aggregate"

	"go.uber.org/mock/gomock"
)

func TestHandleLedgerOutboxEventProjectsTransactionFromNumericIDMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	projector := ledgerprojection.NewMockProjector(ctrl)

	handler := &messageHandler{
		projector: projector,
	}

	projectedAt := time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC)
	messageValue := mustMarshalOutboxMessage(t, outboxMessage{
		ID:          mustMarshalRawMessage(t, 101),
		AggregateID: "tx-1",
		EventName:   ledgeraggregate.EventNameLedgerAccountTransferredToAccount,
		EventData: mustMarshalRawMessage(t, ledgerprojection.LedgerTransactionProjected{
			TransactionID: "tx-1",
			ReferenceType: "ledger.transfer_to_account",
			ReferenceID:   "tx-1",
			Currency:      "VND",
			CreatedAt:     projectedAt,
			Entries: []ledgerprojection.LedgerTransactionEntry{
				{
					AccountID: "acc-1",
					Currency:  "VND",
					Amount:    -100,
					CreatedAt: projectedAt,
				},
				{
					AccountID: "acc-2",
					Currency:  "VND",
					Amount:    100,
					CreatedAt: projectedAt,
				},
			},
		}),
	})

	projector.EXPECT().
		ProjectTransaction(gomock.Any(), &ledgerprojection.LedgerTransactionProjected{
			TransactionID: "tx-1",
			ReferenceType: "ledger.transfer_to_account",
			ReferenceID:   "tx-1",
			Currency:      "VND",
			CreatedAt:     projectedAt,
			Entries: []ledgerprojection.LedgerTransactionEntry{
				{
					AccountID: "acc-1",
					Currency:  "VND",
					Amount:    -100,
					CreatedAt: projectedAt,
				},
				{
					AccountID: "acc-2",
					Currency:  "VND",
					Amount:    100,
					CreatedAt: projectedAt,
				},
			},
		}).
		Return(nil)

	if err := handler.handleLedgerOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandleLedgerOutboxEventSupportsStringEncodedPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	projector := ledgerprojection.NewMockProjector(ctrl)

	handler := &messageHandler{
		projector: projector,
	}

	projectedAt := time.Date(2026, 4, 16, 11, 5, 0, 0, time.UTC)
	innerPayload, err := json.Marshal(ledgerprojection.LedgerTransactionProjected{
		TransactionID: "tx-2",
		ReferenceType: "payment.succeeded",
		ReferenceID:   "pay-1",
		Currency:      "USD",
		CreatedAt:     projectedAt,
		Entries: []ledgerprojection.LedgerTransactionEntry{
			{
				AccountID: "ledger:clearing:provider:stripe",
				Currency:  "USD",
				Amount:    -42,
				CreatedAt: projectedAt,
			},
			{
				AccountID: "wallet:available",
				Currency:  "USD",
				Amount:    42,
				CreatedAt: projectedAt,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal inner payload failed: %v", err)
	}

	messageValue := mustMarshalOutboxMessage(t, outboxMessage{
		ID:          mustMarshalRawMessage(t, 202),
		AggregateID: "tx-2",
		EventName:   ledgeraggregate.EventNameLedgerAccountReceivedTransfer,
		EventData:   mustMarshalRawMessage(t, string(innerPayload)),
	})

	projector.EXPECT().
		ProjectTransaction(gomock.Any(), &ledgerprojection.LedgerTransactionProjected{
			TransactionID: "tx-2",
			ReferenceType: "payment.succeeded",
			ReferenceID:   "pay-1",
			Currency:      "USD",
			CreatedAt:     projectedAt,
			Entries: []ledgerprojection.LedgerTransactionEntry{
				{
					AccountID: "ledger:clearing:provider:stripe",
					Currency:  "USD",
					Amount:    -42,
					CreatedAt: projectedAt,
				},
				{
					AccountID: "wallet:available",
					Currency:  "USD",
					Amount:    42,
					CreatedAt: projectedAt,
				},
			},
		}).
		Return(nil)

	if err := handler.handleLedgerOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
