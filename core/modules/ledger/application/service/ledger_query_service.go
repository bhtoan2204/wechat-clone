package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ledgerout "wechat-clone/core/modules/ledger/application/dto/out"
	ledgerprojection "wechat-clone/core/modules/ledger/application/projection"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"
)

type LedgerQueryService interface {
	GetAccountBalance(ctx context.Context, accountID, currency string) (*ledgerout.AccountBalanceResponse, error)
	GetTransaction(ctx context.Context, transactionID string) (*ledgerout.TransactionResponse, error)
	ListTransactions(ctx context.Context, accountID, cursor, currency string, limit int) (*ledgerout.ListTransactionResponse, error)
}

type ledgerQueryService struct {
	readRepo ledgerprojection.ReadRepository
}

func NewLedgerQueryService(readRepo ledgerprojection.ReadRepository) LedgerQueryService {
	return &ledgerQueryService{readRepo: readRepo}
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

	balance, err := s.readRepo.GetBalance(ctx, accountID, currency)
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

	transaction, err := s.readRepo.GetTransaction(ctx, transactionID)
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

func (s *ledgerQueryService) ListTransactions(
	ctx context.Context,
	accountID string,
	cursor string,
	currency string,
	limit int,
) (*ledgerout.ListTransactionResponse, error) {
	accountID = strings.TrimSpace(accountID)
	cursor = strings.TrimSpace(cursor)
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if accountID == "" {
		return nil, stackErr.Error(fmt.Errorf("%v: account_id is required", ErrValidation))
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter := ledgerprojection.ListTransactionsFilter{
		AccountID: accountID,
		Currency:  currency,
		Limit:     limit + 1,
	}

	if cursor != "" {
		createdAt, transactionID, err := utils.DecodeCursor(cursor)
		if err != nil {
			return nil, stackErr.Error(fmt.Errorf("%v: invalid cursor", ErrValidation))
		}
		filter.CursorCreatedAt = &createdAt
		filter.CursorTransactionID = transactionID
	}

	total, err := s.readRepo.CountTransactions(ctx, accountID, currency)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	transactions, err := s.readRepo.ListTransactions(ctx, filter)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	hasMore := false
	if len(transactions) > limit {
		hasMore = true
		transactions = transactions[:limit]
	}

	records := make([]ledgerout.TransactionResponse, 0, len(transactions))
	for _, transaction := range transactions {
		if transaction == nil {
			continue
		}

		entries := make([]ledgerout.LedgerEntryResponse, 0, len(transaction.Entries))
		for _, entry := range transaction.Entries {
			if entry == nil {
				continue
			}
			entries = append(entries, ledgerout.LedgerEntryResponse{
				ID:            entry.ID,
				TransactionID: entry.TransactionID,
				AccountID:     entry.AccountID,
				Currency:      entry.Currency,
				Amount:        entry.Amount,
				CreatedAt:     entry.CreatedAt,
			})
		}

		records = append(records, ledgerout.TransactionResponse{
			TransactionID: transaction.TransactionID,
			Currency:      transaction.Currency,
			CreatedAt:     transaction.CreatedAt,
			Entries:       entries,
		})
	}

	nextCursor := ""
	if hasMore && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor = utils.EncodeCursor(last.CreatedAt.UTC().Format(time.RFC3339Nano), last.TransactionID)
	}

	return &ledgerout.ListTransactionResponse{
		Limit:      limit,
		Size:       len(records),
		Total:      total,
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Records:    records,
	}, nil
}
