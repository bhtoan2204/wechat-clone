package service

import (
	"context"

	ledgerout "wechat-clone/core/modules/ledger/application/dto/out"
	ledgerprojection "wechat-clone/core/modules/ledger/application/projection"
	"wechat-clone/core/modules/ledger/domain/entity"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
)

//go:generate mockgen -package=service -destination=services_mock.go -source=services.go
type Services interface {
	LedgerQueryService
	LedgerService
}

type services struct {
	ledgerQueryService LedgerQueryService
	ledgerService      LedgerService
}

func NewServices(baseRepo ledgerrepos.Repos, readRepo ledgerprojection.ReadRepository) Services {
	ledgerService := NewLedgerService(baseRepo)
	ledgerQueryService := NewLedgerQueryService(readRepo)

	return &services{
		ledgerQueryService: ledgerQueryService,
		ledgerService:      ledgerService,
	}
}

func (s *services) GetAccountBalance(ctx context.Context, accountID, currency string) (*ledgerout.AccountBalanceResponse, error) {
	return s.ledgerQueryService.GetAccountBalance(ctx, accountID, currency)
}

func (s *services) GetTransaction(ctx context.Context, transactionID string) (*ledgerout.TransactionResponse, error) {
	return s.ledgerQueryService.GetTransaction(ctx, transactionID)
}

func (s *services) ListTransactions(ctx context.Context, accountID, cursor, currency string, limit int) (*ledgerout.ListTransactionResponse, error) {
	return s.ledgerQueryService.ListTransactions(ctx, accountID, cursor, currency, limit)
}

func (s *services) TransferToAccount(ctx context.Context, command TransferToAccountCommand) (*entity.LedgerTransaction, error) {
	return s.ledgerService.TransferToAccount(ctx, command)
}

func (s *services) RecordLedgerEvents(ctx context.Context, command RecordLedgerEventsCommand) error {
	return s.ledgerService.RecordLedgerEvents(ctx, command)
}

func (s *services) RecordPaymentSucceeded(ctx context.Context, command RecordPaymentSucceededCommand) error {
	return s.ledgerService.RecordPaymentSucceeded(ctx, command)
}

func (s *services) RecordPaymentReversed(ctx context.Context, command RecordPaymentReversedCommand) error {
	return s.ledgerService.RecordPaymentReversed(ctx, command)
}
