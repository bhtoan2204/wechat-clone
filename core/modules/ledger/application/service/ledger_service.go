package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	ledgerrepo "go-socket/core/modules/ledger/infra/persistent/repository"
	"go-socket/core/shared/pkg/logging"
)

type LedgerService struct {
	baseRepo ledgerrepos.Repos
}

func NewLedgerService(baseRepo ledgerrepos.Repos) *LedgerService {
	return &LedgerService{baseRepo: baseRepo}
}

func (s *LedgerService) CreateTransaction(ctx context.Context, req *ledgerin.CreateTransactionRequest) (*ledgerout.TransactionResponse, error) {
	if err := wrapValidation(req.Validate()); err != nil {
		return nil, err
	}

	var transaction *entity.LedgerTransaction
	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		var err error
		transaction, err = s.createTransaction(ctx, txRepos.LedgerRepository(), req.TransactionID, toLedgerEntryInputs(req.Entries))
		return err
	}); err != nil {
		return nil, err
	}

	return toTransactionResponse(transaction), nil
}

func (s *LedgerService) GetAccountBalance(ctx context.Context, accountID string) (*ledgerout.AccountBalanceResponse, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, fmt.Errorf("%w: account_id is required", ErrValidation)
	}

	balance, err := s.baseRepo.LedgerRepository().GetBalance(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &ledgerout.AccountBalanceResponse{
		AccountID: accountID,
		Balance:   balance,
	}, nil
}

func (s *LedgerService) GetTransaction(ctx context.Context, transactionID string) (*ledgerout.TransactionResponse, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, fmt.Errorf("%w: transaction_id is required", ErrValidation)
	}

	transaction, err := s.baseRepo.LedgerRepository().GetTransaction(ctx, transactionID)
	if errors.Is(err, ledgerrepo.ErrNotFound) {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, transactionID)
	}
	if err != nil {
		return nil, err
	}

	return toTransactionResponse(transaction), nil
}

func (s *LedgerService) createTransaction(ctx context.Context, repo ledgerrepos.LedgerRepository, transactionID string, entries []entity.LedgerEntryInput) (*entity.LedgerTransaction, error) {
	if err := validateLedgerInputs(transactionID, entries); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	transaction := &entity.LedgerTransaction{
		TransactionID: strings.TrimSpace(transactionID),
		CreatedAt:     now,
	}
	if err := repo.CreateTransaction(ctx, transaction); err != nil {
		if errors.Is(err, ledgerrepo.ErrDuplicate) {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateTransaction, transaction.TransactionID)
		}
		return nil, err
	}

	ledgerEntries := make([]*entity.LedgerEntry, 0, len(entries))
	for _, entry := range entries {
		ledgerEntries = append(ledgerEntries, &entity.LedgerEntry{
			TransactionID: transaction.TransactionID,
			AccountID:     strings.TrimSpace(entry.AccountID),
			Amount:        entry.Amount,
			CreatedAt:     now,
		})
	}
	if err := repo.InsertEntries(ctx, ledgerEntries); err != nil {
		return nil, err
	}

	transaction, err := repo.GetTransaction(ctx, transaction.TransactionID)
	if errors.Is(err, ledgerrepo.ErrNotFound) {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, transactionID)
	}
	if err != nil {
		return nil, err
	}

	logging.FromContext(ctx).Infow("ledger transaction created",
		"transaction_id", transaction.TransactionID,
		"entries_count", len(transaction.Entries),
	)

	return transaction, nil
}

func validateLedgerInputs(transactionID string, entries []entity.LedgerEntryInput) error {
	if strings.TrimSpace(transactionID) == "" {
		return fmt.Errorf("%w: transaction_id is required", ErrValidation)
	}
	if len(entries) < 2 {
		return fmt.Errorf("%w: at least 2 entries are required", ErrValidation)
	}

	var sum int64
	for idx, entry := range entries {
		if strings.TrimSpace(entry.AccountID) == "" {
			return fmt.Errorf("%w: entries[%d].account_id is required", ErrValidation, idx)
		}
		if entry.Amount == 0 {
			return fmt.Errorf("%w: entries[%d].amount must be non-zero", ErrValidation, idx)
		}
		sum += entry.Amount
	}

	if sum != 0 {
		return fmt.Errorf("%w: entries must balance to zero", ErrValidation)
	}
	return nil
}

func toLedgerEntryInputs(entries []ledgerin.LedgerEntryInput) []entity.LedgerEntryInput {
	out := make([]entity.LedgerEntryInput, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entity.LedgerEntryInput{
			AccountID: strings.TrimSpace(entry.AccountID),
			Amount:    entry.Amount,
		})
	}
	return out
}

func toTransactionResponse(transaction *entity.LedgerTransaction) *ledgerout.TransactionResponse {
	entries := make([]ledgerout.LedgerEntryResponse, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		entries = append(entries, ledgerout.LedgerEntryResponse{
			ID:            entry.ID,
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}

	return &ledgerout.TransactionResponse{
		TransactionID: transaction.TransactionID,
		CreatedAt:     transaction.CreatedAt,
		Entries:       entries,
	}
}

func wrapValidation(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrValidation, err.Error())
}
