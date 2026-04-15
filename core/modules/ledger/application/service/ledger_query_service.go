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

type LedgerQueryService interface {
	GetAccountBalance(ctx context.Context, accountID, currency string) (*ledgerout.AccountBalanceResponse, error)
	GetTransaction(ctx context.Context, transactionID string) (*ledgerout.TransactionResponse, error)
}

type ledgerQueryService struct {
	baseRepo ledgerrepos.Repos
}

func NewLedgerQueryService(baseRepo ledgerrepos.Repos) LedgerQueryService {
	return &ledgerQueryService{baseRepo: baseRepo}
}

func (s *ledgerQueryService) GetAccountBalance(ctx context.Context, accountID, currency string) (*ledgerout.AccountBalanceResponse, error) {
	accountID = strings.TrimSpace(accountID)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if accountID == "" {
		return nil, stackErr.Error(fmt.Errorf("%v: account_id is required", ErrValidation))
	}
	if currency == "" {
		return nil, stackErr.Error(fmt.Errorf("%v: currency is required", ErrValidation))
	}

	balance, err := s.baseRepo.LedgerRepository().GetBalance(ctx, accountID, currency)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &ledgerout.AccountBalanceResponse{
		AccountID: accountID,
		Currency:  currency,
		Balance:   balance,
	}, nil
}

func (s *ledgerQueryService) GetTransaction(ctx context.Context, transactionID string) (*ledgerout.TransactionResponse, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, stackErr.Error(fmt.Errorf("%v: transaction_id is required", ErrValidation))
	}

	transaction, err := s.baseRepo.LedgerRepository().GetTransaction(ctx, transactionID)
	if errors.Is(err, ledgerrepos.ErrNotFound) {
		return nil, stackErr.Error(fmt.Errorf("%v: %s", ErrTransactionNotFound, transactionID))
	}
	if err != nil {
		return nil, stackErr.Error(err)
	}

	entries := make([]ledgerout.LedgerEntryResponse, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		entries = append(entries, ledgerout.LedgerEntryResponse{
			ID:            entry.ID,
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Currency:      entry.Currency,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}

	return &ledgerout.TransactionResponse{
		TransactionID: transaction.TransactionID,
		Currency:      transaction.Currency,
		CreatedAt:     transaction.CreatedAt,
		Entries:       entries,
	}, nil
}
