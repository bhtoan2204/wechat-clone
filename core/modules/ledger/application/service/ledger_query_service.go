package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/shared/pkg/stackErr"
)

type LedgerQueryService struct {
	baseRepo ledgerrepos.Repos
}

func NewLedgerQueryService(baseRepo ledgerrepos.Repos) *LedgerQueryService {
	return &LedgerQueryService{baseRepo: baseRepo}
}

func (s *LedgerQueryService) GetAccountBalance(ctx context.Context, accountID string) (*ledgerout.AccountBalanceResponse, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, stackErr.Error(fmt.Errorf("%w: account_id is required", ErrValidation))
	}

	balance, err := s.baseRepo.LedgerRepository().GetBalance(ctx, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &ledgerout.AccountBalanceResponse{
		AccountID: accountID,
		Balance:   balance,
	}, nil
}

func (s *LedgerQueryService) GetTransaction(ctx context.Context, transactionID string) (*ledgerout.TransactionResponse, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, stackErr.Error(fmt.Errorf("%w: transaction_id is required", ErrValidation))
	}

	transaction, err := s.baseRepo.LedgerRepository().GetTransaction(ctx, transactionID)
	if errors.Is(err, ledgerrepos.ErrNotFound) {
		return nil, stackErr.Error(fmt.Errorf("%w: %s", ErrTransactionNotFound, transactionID))
	}
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return toTransactionResponse(transaction), nil
}
