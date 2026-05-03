package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ledgerprojection "wechat-clone/core/modules/ledger/application/projection"
	"wechat-clone/core/modules/ledger/domain/entity"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	"wechat-clone/core/shared/utils"

	"go.uber.org/mock/gomock"
)

type concurrentLedgerReadRepo struct {
	countStarted chan struct{}
	listStarted  chan struct{}

	countErr error
	listErr  error
}

func newConcurrentLedgerReadRepo() *concurrentLedgerReadRepo {
	return &concurrentLedgerReadRepo{
		countStarted: make(chan struct{}),
		listStarted:  make(chan struct{}),
	}
}

func (r *concurrentLedgerReadRepo) ProjectTransaction(context.Context, *ledgerprojection.LedgerTransactionProjected) error {
	return nil
}

func (r *concurrentLedgerReadRepo) GetBalance(context.Context, string, string) (int64, error) {
	return 0, nil
}

func (r *concurrentLedgerReadRepo) GetTransaction(context.Context, string) (*entity.LedgerTransaction, error) {
	return nil, nil
}

func (r *concurrentLedgerReadRepo) CountTransactions(ctx context.Context, _, _ string) (int64, error) {
	close(r.countStarted)
	select {
	case <-r.listStarted:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	if r.countErr != nil {
		return 0, r.countErr
	}
	return 2, nil
}

func (r *concurrentLedgerReadRepo) ListTransactions(ctx context.Context, _ ledgerprojection.ListTransactionsFilter) ([]*entity.LedgerTransaction, error) {
	close(r.listStarted)
	select {
	case <-r.countStarted:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if r.listErr != nil {
		return nil, r.listErr
	}
	return []*entity.LedgerTransaction{{TransactionID: "tx-1", Currency: "VND", CreatedAt: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)}}, nil
}

func TestLedgerQueryServiceListTransactions(t *testing.T) {
	t.Run("maps paginated account transactions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		readRepo := ledgerprojection.NewMockReadRepository(ctrl)

		readRepo.EXPECT().CountTransactions(gomock.Any(), "acc-1", "VND").Return(int64(3), nil)
		readRepo.EXPECT().
			ListTransactions(gomock.Any(), ledgerprojection.ListTransactionsFilter{
				AccountID: "acc-1",
				Currency:  "VND",
				Limit:     3,
			}).
			Return([]*entity.LedgerTransaction{
				{
					TransactionID: "tx-3",
					Currency:      "VND",
					CreatedAt:     time.Date(2026, 4, 16, 10, 2, 0, 0, time.UTC),
					Entries: []*entity.LedgerEntry{
						{TransactionID: "tx-3", AccountID: "acc-1", Currency: "VND", Amount: 300, CreatedAt: time.Date(2026, 4, 16, 10, 2, 0, 0, time.UTC)},
					},
				},
				{
					TransactionID: "tx-2",
					Currency:      "VND",
					CreatedAt:     time.Date(2026, 4, 16, 10, 1, 0, 0, time.UTC),
					Entries: []*entity.LedgerEntry{
						{TransactionID: "tx-2", AccountID: "acc-1", Currency: "VND", Amount: 200, CreatedAt: time.Date(2026, 4, 16, 10, 1, 0, 0, time.UTC)},
					},
				},
				{
					TransactionID: "tx-1",
					Currency:      "VND",
					CreatedAt:     time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
					Entries: []*entity.LedgerEntry{
						{TransactionID: "tx-1", AccountID: "acc-1", Currency: "VND", Amount: 100, CreatedAt: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)},
					},
				},
			}, nil)

		queryService := NewLedgerQueryService(readRepo)
		response, err := queryService.ListTransactions(context.Background(), "acc-1", "", "vnd", 2)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if response.Limit != 2 {
			t.Fatalf("expected limit 2, got %d", response.Limit)
		}
		if response.Size != 2 {
			t.Fatalf("expected size 2, got %d", response.Size)
		}
		if response.Total != 3 {
			t.Fatalf("expected total 3, got %d", response.Total)
		}
		if !response.HasMore {
			t.Fatalf("expected has_more true")
		}
		if len(response.Records) != 2 {
			t.Fatalf("expected 2 records, got %d", len(response.Records))
		}
		if response.Records[0].TransactionID != "tx-3" {
			t.Fatalf("expected first transaction tx-3, got %s", response.Records[0].TransactionID)
		}
		if response.Records[1].TransactionID != "tx-2" {
			t.Fatalf("expected second transaction tx-2, got %s", response.Records[1].TransactionID)
		}
		if response.NextCursor == "" {
			t.Fatalf("expected next cursor")
		}

		createdAt, transactionID, err := utils.DecodeCursor(response.NextCursor)
		if err != nil {
			t.Fatalf("decode next cursor failed: %v", err)
		}
		if transactionID != "tx-2" {
			t.Fatalf("expected cursor transaction tx-2, got %s", transactionID)
		}
		if !createdAt.Equal(time.Date(2026, 4, 16, 10, 1, 0, 0, time.UTC)) {
			t.Fatalf("unexpected cursor created_at: %s", createdAt)
		}
	})

	t.Run("rejects invalid cursor", func(t *testing.T) {
		queryService := NewLedgerQueryService(nil)

		_, err := queryService.ListTransactions(context.Background(), "acc-1", "not-base64", "", 20)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "invalid cursor") {
			t.Fatalf("expected invalid cursor error, got %v", err)
		}
	})
}

func TestLedgerQueryServiceLoadTransactionPage(t *testing.T) {
	t.Run("loads count and page concurrently", func(t *testing.T) {
		readRepo := newConcurrentLedgerReadRepo()
		queryService := &ledgerQueryService{readRepo: readRepo}

		total, transactions, err := queryService.loadTransactionPage(context.Background(), "acc-1", "VND", ledgerprojection.ListTransactionsFilter{
			AccountID: "acc-1",
			Currency:  "VND",
			Limit:     20,
		})
		if err != nil {
			t.Fatalf("loadTransactionPage() error = %v", err)
		}
		if total != 2 {
			t.Fatalf("expected total 2, got %d", total)
		}
		if len(transactions) != 1 || transactions[0].TransactionID != "tx-1" {
			t.Fatalf("unexpected transactions: %#v", transactions)
		}
	})

	t.Run("returns count error", func(t *testing.T) {
		readRepo := newConcurrentLedgerReadRepo()
		readRepo.countErr = errors.New("count failed")
		queryService := &ledgerQueryService{readRepo: readRepo}

		_, _, err := queryService.loadTransactionPage(context.Background(), "acc-1", "VND", ledgerprojection.ListTransactionsFilter{
			AccountID: "acc-1",
			Currency:  "VND",
			Limit:     20,
		})
		if err == nil || !strings.Contains(err.Error(), "count failed") {
			t.Fatalf("expected count error, got %v", err)
		}
	})

	t.Run("returns list error", func(t *testing.T) {
		readRepo := newConcurrentLedgerReadRepo()
		readRepo.listErr = errors.New("list failed")
		queryService := &ledgerQueryService{readRepo: readRepo}

		_, _, err := queryService.loadTransactionPage(context.Background(), "acc-1", "VND", ledgerprojection.ListTransactionsFilter{
			AccountID: "acc-1",
			Currency:  "VND",
			Limit:     20,
		})
		if err == nil || !strings.Contains(err.Error(), "list failed") {
			t.Fatalf("expected list error, got %v", err)
		}
	})
}

func TestLedgerQueryServiceGetAccountBalanceReadsProjectedBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	readRepo := ledgerprojection.NewMockReadRepository(ctrl)

	readRepo.EXPECT().GetBalance(gomock.Any(), "acc-1", "VND").Return(int64(777), nil)

	queryService := NewLedgerQueryService(readRepo)
	response, err := queryService.GetAccountBalance(context.Background(), " acc-1 ", " vnd ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if response.AccountID != "acc-1" {
		t.Fatalf("unexpected account id: %s", response.AccountID)
	}
	if response.Currency != "VND" {
		t.Fatalf("unexpected currency: %s", response.Currency)
	}
	if response.Balance != 777 {
		t.Fatalf("unexpected balance: %d", response.Balance)
	}
}

func TestLedgerQueryServiceGetTransaction(t *testing.T) {
	t.Run("maps transaction entries", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		readRepo := ledgerprojection.NewMockReadRepository(ctrl)
		now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

		readRepo.EXPECT().GetTransaction(gomock.Any(), "tx-1").Return(&entity.LedgerTransaction{
			TransactionID: "tx-1",
			Currency:      "VND",
			CreatedAt:     now,
			Entries: []*entity.LedgerEntry{
				{ID: 1, TransactionID: "tx-1", AccountID: "acc-1", Currency: "VND", Amount: 100, CreatedAt: now},
			},
		}, nil)

		queryService := NewLedgerQueryService(readRepo)
		response, err := queryService.GetTransaction(context.Background(), " tx-1 ")
		if err != nil {
			t.Fatalf("GetTransaction() error = %v", err)
		}
		if response.TransactionID != "tx-1" || response.Currency != "VND" {
			t.Fatalf("unexpected response: %#v", response)
		}
		if len(response.Entries) != 1 || response.Entries[0].ID != 1 {
			t.Fatalf("unexpected entries: %#v", response.Entries)
		}
	})

	t.Run("rejects blank transaction id", func(t *testing.T) {
		queryService := NewLedgerQueryService(nil)

		_, err := queryService.GetTransaction(context.Background(), " ")
		if err == nil || !strings.Contains(err.Error(), "transaction_id is required") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("maps not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		readRepo := ledgerprojection.NewMockReadRepository(ctrl)

		readRepo.EXPECT().GetTransaction(gomock.Any(), "missing-tx").Return(nil, ledgerrepos.ErrNotFound)

		queryService := NewLedgerQueryService(readRepo)
		_, err := queryService.GetTransaction(context.Background(), "missing-tx")
		if err == nil || !strings.Contains(err.Error(), ErrTransactionNotFound.Error()) {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}
