package service

import (
	"context"
	"testing"
	"time"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	paymententity "wechat-clone/core/modules/payment/domain/entity"
	sharedevents "wechat-clone/core/shared/contracts/events"
	eventpkg "wechat-clone/core/shared/pkg/event"

	"go.uber.org/mock/gomock"
)

func TestPaymentEventServiceHandleWithdrawalRequestedRecordsPrincipalAndFeeEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	ledgerSvc := NewMockLedgerService(ctrl)
	service := &paymentEventService{ledgerService: ledgerSvc, feeAccountID: "ledger:fees"}
	now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	ledgerSvc.EXPECT().
		RecordLedgerEvents(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, command RecordLedgerEventsCommand) error {
			assertLedgerEventNames(t, command.Events,
				ledgeraggregate.EventNameLedgerAccountReserveWithdrawal,
				ledgeraggregate.EventNameLedgerAccountReceiveWithdrawalHold,
				ledgeraggregate.EventNameLedgerAccountReserveWithdrawal,
				ledgeraggregate.EventNameLedgerAccountReceiveWithdrawalHold,
			)
			assertAggregateIDs(t, command.Events, "wallet:available", "ledger:clearing:provider:stripe", "wallet:available", "ledger:fees")
			return nil
		})

	if err := service.HandleWithdrawalRequested(context.Background(), sharedevents.PaymentWithdrawalRequestedEvent{
		PaymentID:          "pay-1",
		Provider:           "stripe",
		DebitAccountID:     "wallet:available",
		Amount:             100,
		FeeAmount:          5,
		Currency:           "VND",
		RequestedAt:        now,
		ClearingAccountKey: "",
	}); err != nil {
		t.Fatalf("HandleWithdrawalRequested() error = %v", err)
	}
}

func TestPaymentEventServiceHandleSucceeded(t *testing.T) {
	t.Run("records top-up principal and fee events", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ledgerSvc := NewMockLedgerService(ctrl)
		service := &paymentEventService{ledgerService: ledgerSvc, feeAccountID: "ledger:fees"}
		now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

		ledgerSvc.EXPECT().
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command RecordLedgerEventsCommand) error {
				assertLedgerEventNames(t, command.Events,
					ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent,
					ledgeraggregate.EventNameLedgerAccountDepositFromIntent,
					ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent,
					ledgeraggregate.EventNameLedgerAccountDepositFromIntent,
				)
				assertAggregateIDs(t, command.Events, "ledger:clearing:provider:stripe", "wallet:available", "ledger:clearing:provider:stripe", "ledger:fees")
				return nil
			})

		if err := service.HandleSucceeded(context.Background(), sharedevents.PaymentSucceededEvent{
			Workflow:           paymententity.PaymentWorkflowTopUp,
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Amount:             100,
			FeeAmount:          5,
			Currency:           "VND",
			SucceededAt:        now,
		}); err != nil {
			t.Fatalf("HandleSucceeded() error = %v", err)
		}
	})

	t.Run("ignores withdrawal success because holds are released by provider outcome flows", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ledgerSvc := NewMockLedgerService(ctrl)
		service := &paymentEventService{ledgerService: ledgerSvc, feeAccountID: "ledger:fees"}

		if err := service.HandleSucceeded(context.Background(), sharedevents.PaymentSucceededEvent{
			Workflow:        paymententity.PaymentWorkflowWithdrawal,
			PaymentID:       "pay-1",
			CreditAccountID: "wallet:available",
			Amount:          100,
			Currency:        "VND",
		}); err != nil {
			t.Fatalf("HandleSucceeded() error = %v", err)
		}
	})
}

func TestPaymentEventServiceHandleFailed(t *testing.T) {
	t.Run("records withdrawal hold release events", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ledgerSvc := NewMockLedgerService(ctrl)
		service := &paymentEventService{ledgerService: ledgerSvc, feeAccountID: "ledger:fees"}
		now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

		ledgerSvc.EXPECT().
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command RecordLedgerEventsCommand) error {
				assertLedgerEventNames(t, command.Events,
					ledgeraggregate.EventNameLedgerAccountReleaseWithdrawal,
					ledgeraggregate.EventNameLedgerAccountWithdrawReleasedHold,
					ledgeraggregate.EventNameLedgerAccountReleaseWithdrawal,
					ledgeraggregate.EventNameLedgerAccountWithdrawReleasedHold,
				)
				assertAggregateIDs(t, command.Events, "wallet:available", "ledger:clearing:provider:stripe", "wallet:available", "ledger:fees")
				return nil
			})

		if err := service.HandleFailed(context.Background(), sharedevents.PaymentFailedEvent{
			Workflow:           paymententity.PaymentWorkflowWithdrawal,
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			DebitAccountID:     "wallet:available",
			Amount:             100,
			FeeAmount:          5,
			Currency:           "VND",
			OccurredAt:         now,
		}); err != nil {
			t.Fatalf("HandleFailed() error = %v", err)
		}
	})

	t.Run("ignores top-up failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ledgerSvc := NewMockLedgerService(ctrl)
		service := &paymentEventService{ledgerService: ledgerSvc, feeAccountID: "ledger:fees"}

		if err := service.HandleFailed(context.Background(), sharedevents.PaymentFailedEvent{
			Workflow: paymententity.PaymentWorkflowTopUp,
		}); err != nil {
			t.Fatalf("HandleFailed() error = %v", err)
		}
	})
}

func TestPaymentEventServiceHandleReversals(t *testing.T) {
	t.Run("records refund principal and fee events", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ledgerSvc := NewMockLedgerService(ctrl)
		service := &paymentEventService{ledgerService: ledgerSvc, feeAccountID: "ledger:fees"}
		now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

		ledgerSvc.EXPECT().
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command RecordLedgerEventsCommand) error {
				assertLedgerEventNames(t, command.Events,
					ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund,
					ledgeraggregate.EventNameLedgerAccountDepositFromRefund,
					ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund,
					ledgeraggregate.EventNameLedgerAccountDepositFromRefund,
				)
				assertAggregateIDs(t, command.Events, "wallet:available", "ledger:clearing:provider:stripe", "ledger:fees", "ledger:clearing:provider:stripe")
				return nil
			})

		if err := service.HandleRefunded(context.Background(), sharedevents.PaymentRefundedEvent{
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Amount:             100,
			FeeAmount:          5,
			Currency:           "VND",
			RefundedAt:         now,
		}); err != nil {
			t.Fatalf("HandleRefunded() error = %v", err)
		}
	})

	t.Run("records chargeback events", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ledgerSvc := NewMockLedgerService(ctrl)
		service := &paymentEventService{ledgerService: ledgerSvc}
		now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

		ledgerSvc.EXPECT().
			RecordLedgerEvents(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, command RecordLedgerEventsCommand) error {
				assertLedgerEventNames(t, command.Events,
					ledgeraggregate.EventNameLedgerAccountWithdrawFromChargeback,
					ledgeraggregate.EventNameLedgerAccountDepositFromChargeback,
				)
				assertAggregateIDs(t, command.Events, "wallet:available", "ledger:clearing:provider:stripe")
				return nil
			})

		if err := service.HandleChargeback(context.Background(), sharedevents.PaymentChargebackEvent{
			PaymentID:          "pay-1",
			ClearingAccountKey: "provider:stripe",
			CreditAccountID:    "wallet:available",
			Amount:             100,
			Currency:           "VND",
			ChargedBackAt:      now,
		}); err != nil {
			t.Fatalf("HandleChargeback() error = %v", err)
		}
	})
}

func TestPaymentEventServiceHelpers(t *testing.T) {
	if NewPaymentEventService(nil, "ledger:fees") != nil {
		t.Fatalf("expected nil service without base repo")
	}
	if ledgerClearingAccountID(" Provider:Stripe ") != "ledger:clearing:provider:stripe" {
		t.Fatalf("unexpected clearing account id")
	}
	if providerClearingAccountKey(" Stripe ") != "provider:stripe" {
		t.Fatalf("unexpected provider clearing account key")
	}
	if providerClearingAccountKey(" ") != "" {
		t.Fatalf("expected blank provider key")
	}
	if reversalSuffix(sharedevents.EventPaymentRefunded) != "refunded" {
		t.Fatalf("unexpected refund suffix")
	}
	if reversalSuffix(sharedevents.EventPaymentChargeback) != "chargeback" {
		t.Fatalf("unexpected chargeback suffix")
	}
	if reversalSuffix("other") != "reversed" {
		t.Fatalf("unexpected default reversal suffix")
	}
	if debitLedgerEventNameForReversal(sharedevents.EventPaymentRefunded) != ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund {
		t.Fatalf("unexpected refund debit event")
	}
	if debitLedgerEventNameForReversal(sharedevents.EventPaymentChargeback) != ledgeraggregate.EventNameLedgerAccountWithdrawFromChargeback {
		t.Fatalf("unexpected chargeback debit event")
	}
	if debitLedgerEventNameForReversal("other") != "" {
		t.Fatalf("unexpected default debit event")
	}
	if creditLedgerEventNameForReversal(sharedevents.EventPaymentRefunded) != ledgeraggregate.EventNameLedgerAccountDepositFromRefund {
		t.Fatalf("unexpected refund credit event")
	}
	if creditLedgerEventNameForReversal(sharedevents.EventPaymentChargeback) != ledgeraggregate.EventNameLedgerAccountDepositFromChargeback {
		t.Fatalf("unexpected chargeback credit event")
	}
	if creditLedgerEventNameForReversal("other") != "" {
		t.Fatalf("unexpected default credit event")
	}
	if resolvePaymentEventID(" pay-1 ", "tx-1") != "pay-1" {
		t.Fatalf("expected payment id to win")
	}
	if resolvePaymentEventID(" ", " tx-1 ") != "tx-1" {
		t.Fatalf("expected transaction id fallback")
	}
}

func assertLedgerEventNames(t *testing.T, events []eventpkg.Event, expected ...string) {
	t.Helper()
	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d: %#v", len(expected), len(events), events)
	}
	for i, expectedName := range expected {
		if events[i].EventName != expectedName {
			t.Fatalf("event[%d] name = %s, want %s", i, events[i].EventName, expectedName)
		}
	}
}

func assertAggregateIDs(t *testing.T, events []eventpkg.Event, expected ...string) {
	t.Helper()
	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d: %#v", len(expected), len(events), events)
	}
	for i, expectedID := range expected {
		if events[i].AggregateID != expectedID {
			t.Fatalf("event[%d] aggregate_id = %s, want %s", i, events[i].AggregateID, expectedID)
		}
	}
}
