package service

import (
	"context"

	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
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

func NewServices(baseRepo ledgerrepos.Repos) Services {
	ledgerService := NewLedgerService(baseRepo)
	ledgerQueryService := NewLedgerQueryService(baseRepo)

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

func (s *services) TransferToAccount(ctx context.Context, command TransferToAccountCommand) (*entity.LedgerTransaction, error) {
	return s.ledgerService.TransferToAccount(ctx, command)
}

func (s *services) RecordPaymentSucceeded(ctx context.Context, command RecordPaymentSucceededCommand) error {
	return s.ledgerService.RecordPaymentSucceeded(ctx, command)
}
