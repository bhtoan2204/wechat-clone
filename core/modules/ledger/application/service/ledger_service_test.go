package service

import (
	"context"
	"errors"
	"testing"
	"time"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	ledgerrepo "go-socket/core/modules/ledger/infra/persistent/repository"
)

func TestLedgerServiceCreateTransaction(t *testing.T) {
	t.Run("valid transaction", func(t *testing.T) {
		repos := newFakeRepos()
		service := NewLedgerService(repos)

		response, err := service.CreateTransaction(context.Background(), &ledgerin.CreateTransactionRequest{
			TransactionID: "ledger-tx-valid",
			Entries: []ledgerin.LedgerEntryInput{
				{AccountID: "acc-a", Amount: -100},
				{AccountID: "acc-b", Amount: 100},
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if response.TransactionID != "ledger-tx-valid" {
			t.Fatalf("expected transaction id ledger-tx-valid, got %s", response.TransactionID)
		}
		if len(response.Entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(response.Entries))
		}
	})

	t.Run("invalid transaction", func(t *testing.T) {
		repos := newFakeRepos()
		service := NewLedgerService(repos)

		_, err := service.CreateTransaction(context.Background(), &ledgerin.CreateTransactionRequest{
			TransactionID: "ledger-tx-invalid",
			Entries: []ledgerin.LedgerEntryInput{
				{AccountID: "acc-a", Amount: -100},
				{AccountID: "acc-b", Amount: 50},
			},
		})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("duplicate transaction", func(t *testing.T) {
		repos := newFakeRepos()
		repos.ledgerRepo.transactions["ledger-tx-dup"] = &entity.LedgerTransaction{
			TransactionID: "ledger-tx-dup",
			CreatedAt:     time.Now().UTC(),
		}
		service := NewLedgerService(repos)

		_, err := service.CreateTransaction(context.Background(), &ledgerin.CreateTransactionRequest{
			TransactionID: "ledger-tx-dup",
			Entries: []ledgerin.LedgerEntryInput{
				{AccountID: "acc-a", Amount: -100},
				{AccountID: "acc-b", Amount: 100},
			},
		})
		if !errors.Is(err, ErrDuplicateTransaction) {
			t.Fatalf("expected duplicate transaction error, got %v", err)
		}
	})
}

type fakeRepos struct {
	ledgerRepo *fakeLedgerRepo
}

func newFakeRepos() *fakeRepos {
	return &fakeRepos{
		ledgerRepo: &fakeLedgerRepo{
			nextEntryID:  1,
			transactions: make(map[string]*entity.LedgerTransaction),
		},
	}
}

func (r *fakeRepos) LedgerRepository() ledgerrepos.LedgerRepository {
	return r.ledgerRepo
}

func (r *fakeRepos) PaymentRepository() ledgerrepos.PaymentRepository {
	return nil
}

func (r *fakeRepos) WithTransaction(_ context.Context, fn func(ledgerrepos.Repos) error) error {
	return fn(r)
}

type fakeLedgerRepo struct {
	nextEntryID  int64
	transactions map[string]*entity.LedgerTransaction
}

func (r *fakeLedgerRepo) CreateTransaction(_ context.Context, transaction *entity.LedgerTransaction) error {
	if _, exists := r.transactions[transaction.TransactionID]; exists {
		return ledgerrepo.ErrDuplicate
	}
	r.transactions[transaction.TransactionID] = &entity.LedgerTransaction{
		TransactionID: transaction.TransactionID,
		CreatedAt:     transaction.CreatedAt,
	}
	return nil
}

func (r *fakeLedgerRepo) InsertEntries(_ context.Context, entries []*entity.LedgerEntry) error {
	for _, entry := range entries {
		stored := *entry
		stored.ID = r.nextEntryID
		r.nextEntryID++
		r.transactions[stored.TransactionID].Entries = append(r.transactions[stored.TransactionID].Entries, &stored)
	}
	return nil
}

func (r *fakeLedgerRepo) GetBalance(_ context.Context, accountID string) (int64, error) {
	var balance int64
	for _, transaction := range r.transactions {
		for _, entry := range transaction.Entries {
			if entry.AccountID == accountID {
				balance += entry.Amount
			}
		}
	}
	return balance, nil
}

func (r *fakeLedgerRepo) GetTransaction(_ context.Context, transactionID string) (*entity.LedgerTransaction, error) {
	transaction, exists := r.transactions[transactionID]
	if !exists {
		return nil, ledgerrepo.ErrNotFound
	}
	return transaction, nil
}
