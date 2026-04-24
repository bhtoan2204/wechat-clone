package messaging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	ledgerservice "wechat-clone/core/modules/ledger/application/service"
	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	sharedevents "wechat-clone/core/shared/contracts/events"
	sharedlock "wechat-clone/core/shared/infra/lock"
	eventpkg "wechat-clone/core/shared/pkg/event"

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

	messageValue := mustMarshalOutboxMessage(t, paymentOutboxMessage{
		AggregateID: "pay-aggregate",
		EventName:   sharedevents.EventPaymentSucceeded,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentSucceededEvent{
			PaymentID:          "pay-1",
			TransactionID:      "txn-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
			SucceededAt:        time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
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
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command ledgerservice.RecordLedgerEventsCommand) error {
				assertLedgerPaymentEvent(t, command.Events, 0, "ledger:clearing:provider:stripe", ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent)
				assertLedgerPaymentEvent(t, command.Events, 1, "wallet:available", ledgeraggregate.EventNameLedgerAccountDepositFromIntent)
				return nil
			}),
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
		SucceededAt:        time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal inner payload failed: %v", err)
	}

	messageValue := mustMarshalOutboxMessage(t, paymentOutboxMessage{
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
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command ledgerservice.RecordLedgerEventsCommand) error {
				assertLedgerPaymentEvent(t, command.Events, 0, "ledger:clearing:provider:stripe", ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent)
				assertLedgerPaymentEvent(t, command.Events, 1, "wallet:available", ledgeraggregate.EventNameLedgerAccountDepositFromIntent)
				return nil
			}),
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

	messageValue := mustMarshalOutboxMessage(t, paymentOutboxMessage{
		AggregateID: "pay-aggregate",
		EventName:   sharedevents.EventPaymentRefunded,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentRefundedEvent{
			PaymentID:          "pay-1",
			TransactionID:      "txn-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
			RefundedAt:         time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC),
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
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command ledgerservice.RecordLedgerEventsCommand) error {
				assertLedgerPaymentEvent(t, command.Events, 0, "wallet:available", ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund)
				assertLedgerPaymentEvent(t, command.Events, 1, "ledger:clearing:provider:stripe", ledgeraggregate.EventNameLedgerAccountDepositFromRefund)
				return nil
			}),
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

	messageValue := mustMarshalOutboxMessage(t, paymentOutboxMessage{
		AggregateID: "pay-aggregate",
		EventName:   sharedevents.EventPaymentChargeback,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentChargebackEvent{
			PaymentID:          "pay-1",
			TransactionID:      "txn-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Currency:           "VND",
			Amount:             100,
			ChargedBackAt:      time.Date(2026, 4, 22, 13, 0, 0, 0, time.UTC),
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
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command ledgerservice.RecordLedgerEventsCommand) error {
				assertLedgerPaymentEvent(t, command.Events, 0, "wallet:available", ledgeraggregate.EventNameLedgerAccountWithdrawFromChargeback)
				assertLedgerPaymentEvent(t, command.Events, 1, "ledger:clearing:provider:stripe", ledgeraggregate.EventNameLedgerAccountDepositFromChargeback)
				return nil
			}),
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

func TestHandlePaymentOutboxEventBooksWithdrawalReservationOnRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	locker := sharedlock.NewMockLock(ctrl)
	ledgerService := ledgerservice.NewMockLedgerService(ctrl)

	handler := &messageHandler{
		ledgerService: ledgerService,
		locker:        locker,
		feeAccountID:  "ledger:fee:provider:stripe",
	}

	messageValue := mustMarshalOutboxMessage(t, paymentOutboxMessage{
		AggregateID: "pay-withdrawal-1",
		EventName:   sharedevents.EventPaymentWithdrawalRequested,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentWithdrawalRequestedEvent{
			PaymentID:            "pay-withdrawal-1",
			TransactionID:        "txn-withdrawal-1",
			Provider:             "stripe",
			DebitAccountID:       "wallet:available",
			DestinationAccountID: "acct_1QdrKFJLKs7Pc3Qr",
			Currency:             "VND",
			Amount:               100,
			FeeAmount:            5,
			RequestedAt:          time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
		}),
	})

	gomock.InOrder(
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:fee:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		ledgerService.EXPECT().
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command ledgerservice.RecordLedgerEventsCommand) error {
				assertLedgerPaymentEvent(t, command.Events, 0, "wallet:available", ledgeraggregate.EventNameLedgerAccountReserveWithdrawal)
				assertLedgerPaymentEvent(t, command.Events, 1, "ledger:clearing:provider:stripe", ledgeraggregate.EventNameLedgerAccountReceiveWithdrawalHold)
				assertLedgerPaymentEvent(t, command.Events, 2, "wallet:available", ledgeraggregate.EventNameLedgerAccountReserveWithdrawal)
				assertLedgerPaymentEvent(t, command.Events, 3, "ledger:fee:provider:stripe", ledgeraggregate.EventNameLedgerAccountReceiveWithdrawalHold)
				return nil
			}),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:fee:provider:stripe", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any()).
			Return(true, nil),
	)

	if err := handler.handlePaymentOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandlePaymentOutboxEventReversesWithdrawalReservationOnFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	locker := sharedlock.NewMockLock(ctrl)
	ledgerService := ledgerservice.NewMockLedgerService(ctrl)

	handler := &messageHandler{
		ledgerService: ledgerService,
		locker:        locker,
		feeAccountID:  "ledger:fee:provider:stripe",
	}

	messageValue := mustMarshalOutboxMessage(t, paymentOutboxMessage{
		AggregateID: "pay-withdrawal-2",
		EventName:   sharedevents.EventPaymentFailed,
		EventData: mustMarshalRawMessage(t, sharedevents.PaymentFailedEvent{
			Workflow:           "WITHDRAWAL",
			PaymentID:          "pay-withdrawal-2",
			TransactionID:      "txn-withdrawal-2",
			ClearingAccountKey: "provider:stripe",
			DebitAccountID:     "wallet:available",
			Currency:           "VND",
			Amount:             100,
			FeeAmount:          5,
			OccurredAt:         time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC),
		}),
	})

	gomock.InOrder(
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:ledger:fee:provider:stripe", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		locker.EXPECT().
			AcquireLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).
			Return(true, nil),
		ledgerService.EXPECT().
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command ledgerservice.RecordLedgerEventsCommand) error {
				assertLedgerPaymentEvent(t, command.Events, 0, "wallet:available", ledgeraggregate.EventNameLedgerAccountReleaseWithdrawal)
				assertLedgerPaymentEvent(t, command.Events, 1, "ledger:clearing:provider:stripe", ledgeraggregate.EventNameLedgerAccountWithdrawReleasedHold)
				assertLedgerPaymentEvent(t, command.Events, 2, "wallet:available", ledgeraggregate.EventNameLedgerAccountReleaseWithdrawal)
				assertLedgerPaymentEvent(t, command.Events, 3, "ledger:fee:provider:stripe", ledgeraggregate.EventNameLedgerAccountWithdrawReleasedHold)
				return nil
			}),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:wallet:available", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:fee:provider:stripe", gomock.Any()).
			Return(true, nil),
		locker.EXPECT().
			ReleaseLock(gomock.Any(), "ledger-account:ledger:clearing:provider:stripe", gomock.Any()).
			Return(true, nil),
	)

	if err := handler.handlePaymentOutboxEvent(context.Background(), messageValue); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func mustMarshalOutboxMessage(t *testing.T, message paymentOutboxMessage) []byte {
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

func assertLedgerPaymentEvent(t *testing.T, events []eventpkg.Event, idx int, aggregateID, eventName string) {
	t.Helper()
	if len(events) <= idx {
		t.Fatalf("expected event at index %d, got %d events", idx, len(events))
	}
	if events[idx].AggregateID != aggregateID {
		t.Fatalf("expected aggregate id %s at index %d, got %s", aggregateID, idx, events[idx].AggregateID)
	}
	if events[idx].EventName != eventName {
		t.Fatalf("expected event name %s at index %d, got %s", eventName, idx, events[idx].EventName)
	}
}
