package service

import (
	"context"
	"strings"
	"testing"
	"time"

	ledgerprojection "wechat-clone/core/modules/ledger/application/projection"
	"wechat-clone/core/modules/ledger/domain/entity"
	"wechat-clone/core/shared/utils"

	"go.uber.org/mock/gomock"
)

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
