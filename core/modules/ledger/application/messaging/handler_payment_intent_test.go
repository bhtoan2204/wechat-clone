package messaging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	ledgerservice "wechat-clone/core/modules/ledger/application/service"
	sharedevents "wechat-clone/core/shared/contracts/events"
	sharedlock "wechat-clone/core/shared/infra/lock"

	"go.uber.org/mock/gomock"
)

func TestHandlePaymentOutboxEventLocksPaymentSucceededByAffectedAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	locker := sharedlock.NewMockLock(ctrl)
	ledgerService := ledgerservice.NewMockLedgerService(ctrl)

	handler := &messageHandler{
		ledgerService: ledgerService,
		locker:        locker,
	}

	messageValue := mustMarshalOutboxMessage(t, outboxMessage{
		AggregateID: "pay-aggregate",
		EventName:   sharedevents.EventPaymentSucceeded,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentSucceededEvent{
			PaymentID:          "pay-1",
			TransactionID:      "txn-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
		}),
	})

	gomock.InOrder(
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		ledgerService.EXPECT().
			RecordPaymentSucceeded(gomock.Any(), ledgerservice.RecordPaymentSucceededCommand{
				PaymentID:          "pay-1",
				TransactionID:      "txn-1",
				ClearingAccountKey: "provider:stripe",
				CreditAccountID:    "wallet:available",
				Currency:           "VND",
				Amount:             100,
			}).
			Return(nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any()).
			Return(true, nil),
	)

	if err := handler.handlePaymentOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandlePaymentOutboxEventFallsBackToAggregateIDForCommandPaymentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	locker := sharedlock.NewMockLock(ctrl)
	ledgerService := ledgerservice.NewMockLedgerService(ctrl)

	handler := &messageHandler{
		ledgerService: ledgerService,
		locker:        locker,
	}

	innerPayload, err := json.Marshal(sharedevents.PaymentSucceededEvent{
		TransactionID:      "txn-2",
		ClearingAccountKey: "provider:stripe",
		CreditAccountID:    "wallet:available",
		Currency:           "USD",
		Amount:             42,
	})
	if err != nil {
		t.Fatalf("marshal inner payload failed: %v", err)
	}

	messageValue := mustMarshalOutboxMessage(t, outboxMessage{
		AggregateID: "pay-aggregate-2",
		EventName:   sharedevents.EventPaymentSucceeded,
		EventData:   mustMarshalRawMessage(t, string(innerPayload)),
	})

	gomock.InOrder(
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		ledgerService.EXPECT().
			RecordPaymentSucceeded(gomock.Any(), ledgerservice.RecordPaymentSucceededCommand{
				PaymentID:          "pay-aggregate-2",
				TransactionID:      "txn-2",
				ClearingAccountKey: "provider:stripe",
				CreditAccountID:    "wallet:available",
				Currency:           "USD",
				Amount:             42,
			}).
			Return(nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any()).
			Return(true, nil),
	)

	if err := handler.handlePaymentOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandlePaymentOutboxEventLocksPaymentRefundedByAffectedAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	locker := sharedlock.NewMockLock(ctrl)
	ledgerService := ledgerservice.NewMockLedgerService(ctrl)

	handler := &messageHandler{
		ledgerService: ledgerService,
		locker:        locker,
	}

	messageValue := mustMarshalOutboxMessage(t, outboxMessage{
		AggregateID: "pay-aggregate",
		EventName:   sharedevents.EventPaymentRefunded,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentRefundedEvent{
			PaymentID:          "pay-1",
			TransactionID:      "txn-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
		}),
	})

	gomock.InOrder(
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		ledgerService.EXPECT().
			RecordPaymentReversed(gomock.Any(), ledgerservice.RecordPaymentReversedCommand{
				PaymentID:          "pay-1",
				TransactionID:      "txn-1",
				ClearingAccountKey: "provider:stripe",
				CreditAccountID:    "wallet:available",
				Currency:           "VND",
				Amount:             100,
				ReversalType:       "payment.refunded",
			}).
			Return(nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any()).
			Return(true, nil),
	)

	if err := handler.handlePaymentOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandlePaymentOutboxEventLocksPaymentChargebackByAffectedAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	locker := sharedlock.NewMockLock(ctrl)
	ledgerService := ledgerservice.NewMockLedgerService(ctrl)

	handler := &messageHandler{
		ledgerService: ledgerService,
		locker:        locker,
	}

	messageValue := mustMarshalOutboxMessage(t, outboxMessage{
		AggregateID: "pay-aggregate",
		EventName:   sharedevents.EventPaymentChargeback,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentChargebackEvent{
			PaymentID:          "pay-1",
			TransactionID:      "txn-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
		}),
	})

	gomock.InOrder(
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		ledgerService.EXPECT().
			RecordPaymentReversed(gomock.Any(), ledgerservice.RecordPaymentReversedCommand{
				PaymentID:          "pay-1",
				TransactionID:      "txn-1",
				ClearingAccountKey: "provider:stripe",
				CreditAccountID:    "wallet:available",
				Currency:           "VND",
				Amount:             100,
				ReversalType:       "payment.chargeback",
			}).
			Return(nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any()).
			Return(true, nil),
	)

	if err := handler.handlePaymentOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func mustMarshalOutboxMessage(t *testing.T, message outboxMessage) []byte {
	t.Helper()

	value, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal payment outbox message failed: %v", err)
	}

	return value
}

func mustMarshalRawMessage(t *testing.T, value interface{}) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw message failed: %v", err)
	}

	return raw
}
