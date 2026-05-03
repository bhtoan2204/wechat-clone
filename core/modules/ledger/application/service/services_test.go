package service

import (
	"context"
	"testing"
	"time"

	ledgerout "wechat-clone/core/modules/ledger/application/dto/out"
	"wechat-clone/core/modules/ledger/domain/entity"

	"go.uber.org/mock/gomock"
)

func TestServicesDelegatesToComposedServices(t *testing.T) {
	ledgerQuery := &stubLedgerQueryService{}
	ledgerCommands := &stubLedgerService{}
	services := &services{ledgerQueryService: ledgerQuery, ledgerService: ledgerCommands}
	ctx := context.Background()

	if _, err := services.GetAccountBalance(ctx, "acc-1", "VND"); err != nil {
		t.Fatalf("GetAccountBalance() error = %v", err)
	}
	if ledgerQuery.balanceAccountID != "acc-1" || ledgerQuery.balanceCurrency != "VND" {
		t.Fatalf("GetAccountBalance() did not delegate input")
	}

	if _, err := services.GetTransaction(ctx, "tx-1"); err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}
	if ledgerQuery.transactionID != "tx-1" {
		t.Fatalf("GetTransaction() did not delegate input")
	}

	if _, err := services.ListTransactions(ctx, "acc-1", "cursor", "VND", 10); err != nil {
		t.Fatalf("ListTransactions() error = %v", err)
	}
	if ledgerQuery.listAccountID != "acc-1" || ledgerQuery.listCursor != "cursor" || ledgerQuery.listCurrency != "VND" || ledgerQuery.listLimit != 10 {
		t.Fatalf("ListTransactions() did not delegate input")
	}

	if _, err := services.TransferToAccount(ctx, TransferToAccountCommand{TransactionID: "transfer-1"}); err != nil {
		t.Fatalf("TransferToAccount() error = %v", err)
	}
	if ledgerCommands.transferCommand.TransactionID != "transfer-1" {
		t.Fatalf("TransferToAccount() did not delegate input")
	}

	if err := services.RecordLedgerEvents(ctx, RecordLedgerEventsCommand{}); err != nil {
		t.Fatalf("RecordLedgerEvents() error = %v", err)
	}
	if !ledgerCommands.recordLedgerEventsCalled {
		t.Fatalf("RecordLedgerEvents() was not delegated")
	}

	if err := services.RecordPaymentSucceeded(ctx, RecordPaymentSucceededCommand{PaymentID: "pay-1"}); err != nil {
		t.Fatalf("RecordPaymentSucceeded() error = %v", err)
	}
	if ledgerCommands.paymentSucceededCommand.PaymentID != "pay-1" {
		t.Fatalf("RecordPaymentSucceeded() did not delegate input")
	}

	if err := services.RecordPaymentReversed(ctx, RecordPaymentReversedCommand{PaymentID: "pay-2"}); err != nil {
		t.Fatalf("RecordPaymentReversed() error = %v", err)
	}
	if ledgerCommands.paymentReversedCommand.PaymentID != "pay-2" {
		t.Fatalf("RecordPaymentReversed() did not delegate input")
	}

	if err := services.RecordPaymentReconciliationFailed(ctx, RecordPaymentReconciliationFailedCommand{PaymentID: "pay-3"}); err != nil {
		t.Fatalf("RecordPaymentReconciliationFailed() error = %v", err)
	}
	if ledgerCommands.paymentReconciliationFailedCommand.PaymentID != "pay-3" {
		t.Fatalf("RecordPaymentReconciliationFailed() did not delegate input")
	}
}

func TestGeneratedMockCoverageForLedgerService(t *testing.T) {
	ctrl := gomock.NewController(t)
	ledgerSvc := NewMockLedgerService(ctrl)
	ctx := context.Background()

	ledgerSvc.EXPECT().TransferToAccount(ctx, TransferToAccountCommand{TransactionID: "tx-1"}).Return(&entity.LedgerTransaction{TransactionID: "tx-1"}, nil)
	if _, err := ledgerSvc.TransferToAccount(ctx, TransferToAccountCommand{TransactionID: "tx-1"}); err != nil {
		t.Fatalf("TransferToAccount() error = %v", err)
	}

	ledgerSvc.EXPECT().RecordPaymentSucceeded(ctx, RecordPaymentSucceededCommand{PaymentID: "pay-1"}).Return(nil)
	if err := ledgerSvc.RecordPaymentSucceeded(ctx, RecordPaymentSucceededCommand{PaymentID: "pay-1"}); err != nil {
		t.Fatalf("RecordPaymentSucceeded() error = %v", err)
	}

	ledgerSvc.EXPECT().RecordPaymentReversed(ctx, RecordPaymentReversedCommand{PaymentID: "pay-2"}).Return(nil)
	if err := ledgerSvc.RecordPaymentReversed(ctx, RecordPaymentReversedCommand{PaymentID: "pay-2"}); err != nil {
		t.Fatalf("RecordPaymentReversed() error = %v", err)
	}

	ledgerSvc.EXPECT().RecordPaymentReconciliationFailed(ctx, RecordPaymentReconciliationFailedCommand{PaymentID: "pay-3"}).Return(nil)
	if err := ledgerSvc.RecordPaymentReconciliationFailed(ctx, RecordPaymentReconciliationFailedCommand{PaymentID: "pay-3"}); err != nil {
		t.Fatalf("RecordPaymentReconciliationFailed() error = %v", err)
	}
}

func TestGeneratedMockCoverageForServices(t *testing.T) {
	ctrl := gomock.NewController(t)
	serviceMock := NewMockServices(ctrl)
	ctx := context.Background()

	serviceMock.EXPECT().GetAccountBalance(ctx, "acc-1", "VND").Return(&ledgerout.AccountBalanceResponse{}, nil)
	if _, err := serviceMock.GetAccountBalance(ctx, "acc-1", "VND"); err != nil {
		t.Fatalf("GetAccountBalance() error = %v", err)
	}

	serviceMock.EXPECT().GetTransaction(ctx, "tx-1").Return(&ledgerout.TransactionResponse{}, nil)
	if _, err := serviceMock.GetTransaction(ctx, "tx-1"); err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}

	serviceMock.EXPECT().ListTransactions(ctx, "acc-1", "cursor", "VND", 10).Return(&ledgerout.ListTransactionResponse{}, nil)
	if _, err := serviceMock.ListTransactions(ctx, "acc-1", "cursor", "VND", 10); err != nil {
		t.Fatalf("ListTransactions() error = %v", err)
	}

	serviceMock.EXPECT().RecordLedgerEvents(ctx, RecordLedgerEventsCommand{}).Return(nil)
	if err := serviceMock.RecordLedgerEvents(ctx, RecordLedgerEventsCommand{}); err != nil {
		t.Fatalf("RecordLedgerEvents() error = %v", err)
	}

	serviceMock.EXPECT().RecordPaymentSucceeded(ctx, RecordPaymentSucceededCommand{PaymentID: "pay-1"}).Return(nil)
	if err := serviceMock.RecordPaymentSucceeded(ctx, RecordPaymentSucceededCommand{PaymentID: "pay-1"}); err != nil {
		t.Fatalf("RecordPaymentSucceeded() error = %v", err)
	}

	serviceMock.EXPECT().RecordPaymentReversed(ctx, RecordPaymentReversedCommand{PaymentID: "pay-2"}).Return(nil)
	if err := serviceMock.RecordPaymentReversed(ctx, RecordPaymentReversedCommand{PaymentID: "pay-2"}); err != nil {
		t.Fatalf("RecordPaymentReversed() error = %v", err)
	}

	serviceMock.EXPECT().TransferToAccount(ctx, TransferToAccountCommand{TransactionID: "transfer-1"}).Return(&entity.LedgerTransaction{}, nil)
	if _, err := serviceMock.TransferToAccount(ctx, TransferToAccountCommand{TransactionID: "transfer-1"}); err != nil {
		t.Fatalf("TransferToAccount() error = %v", err)
	}
}

type stubLedgerQueryService struct {
	balanceAccountID string
	balanceCurrency  string
	transactionID    string
	listAccountID    string
	listCursor       string
	listCurrency     string
	listLimit        int
}

func (s *stubLedgerQueryService) GetAccountBalance(_ context.Context, accountID, currency string) (*ledgerout.AccountBalanceResponse, error) {
	s.balanceAccountID = accountID
	s.balanceCurrency = currency
	return &ledgerout.AccountBalanceResponse{AccountID: accountID, Currency: currency}, nil
}

func (s *stubLedgerQueryService) GetTransaction(_ context.Context, transactionID string) (*ledgerout.TransactionResponse, error) {
	s.transactionID = transactionID
	return &ledgerout.TransactionResponse{TransactionID: transactionID}, nil
}

func (s *stubLedgerQueryService) ListTransactions(_ context.Context, accountID, cursor, currency string, limit int) (*ledgerout.ListTransactionResponse, error) {
	s.listAccountID = accountID
	s.listCursor = cursor
	s.listCurrency = currency
	s.listLimit = limit
	return &ledgerout.ListTransactionResponse{Limit: limit}, nil
}

type stubLedgerService struct {
	transferCommand                    TransferToAccountCommand
	recordLedgerEventsCalled           bool
	paymentSucceededCommand            RecordPaymentSucceededCommand
	paymentReversedCommand             RecordPaymentReversedCommand
	paymentReconciliationFailedCommand RecordPaymentReconciliationFailedCommand
}

func (s *stubLedgerService) TransferToAccount(_ context.Context, command TransferToAccountCommand) (*entity.LedgerTransaction, error) {
	s.transferCommand = command
	return &entity.LedgerTransaction{TransactionID: command.TransactionID, CreatedAt: time.Now().UTC()}, nil
}

func (s *stubLedgerService) RecordLedgerEvents(context.Context, RecordLedgerEventsCommand) error {
	s.recordLedgerEventsCalled = true
	return nil
}

func (s *stubLedgerService) RecordPaymentSucceeded(_ context.Context, command RecordPaymentSucceededCommand) error {
	s.paymentSucceededCommand = command
	return nil
}

func (s *stubLedgerService) RecordPaymentReversed(_ context.Context, command RecordPaymentReversedCommand) error {
	s.paymentReversedCommand = command
	return nil
}

func (s *stubLedgerService) RecordPaymentReconciliationFailed(_ context.Context, command RecordPaymentReconciliationFailedCommand) error {
	s.paymentReconciliationFailedCommand = command
	return nil
}
