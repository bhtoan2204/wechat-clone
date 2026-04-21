package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	"wechat-clone/core/modules/ledger/domain/entity"
	eventpkg "wechat-clone/core/shared/pkg/event"
)

type fakeLedgerEventStore struct {
	checkCalled bool
	baseVersion int
	newVersion  int
	checkOK     bool
}

type fakeLedgerOutboxEventsRepo struct{}

func (f *fakeLedgerOutboxEventsRepo) Append(context.Context, eventpkg.Event) error {
	return nil
}

func (f *fakeLedgerEventStore) CreateIfNotExist(context.Context, string, string) error {
	return nil
}

func (f *fakeLedgerEventStore) CheckAndUpdateVersion(_ context.Context, _ string, _ string, baseVersion, newVersion int) (bool, error) {
	f.checkCalled = true
	f.baseVersion = baseVersion
	f.newVersion = newVersion
	return f.checkOK, nil
}

func (f *fakeLedgerEventStore) FindPostedTransaction(context.Context, string, string, string) (*entity.LedgerAccountPosting, error) {
	return nil, nil
}

func (f *fakeLedgerEventStore) ReservePostedTransaction(context.Context, eventpkg.Event) error {
	return nil
}

func (f *fakeLedgerEventStore) Append(context.Context, eventpkg.Event) error {
	return nil
}

func (f *fakeLedgerEventStore) Get(context.Context, string, string, int, eventpkg.Aggregate) error {
	return nil
}

func (f *fakeLedgerEventStore) CreateSnapshot(context.Context, eventpkg.Aggregate) error {
	return nil
}

func (f *fakeLedgerEventStore) ReadSnapshot(context.Context, string, string, eventpkg.Aggregate) (bool, error) {
	return false, nil
}

func TestAggregateStoreSaveUsesExpectedVersion(t *testing.T) {
	repo := &fakeLedgerEventStore{checkOK: true}
	store := &aggregateStoreImpl{repo: repo, outboxRepo: &fakeLedgerOutboxEventsRepo{}}

	aggregate, err := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}
	if _, err := aggregate.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "VND", 300, time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("BookPayment() error = %v", err)
	}

	if err := store.Save(context.Background(), aggregate); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !repo.checkCalled {
		t.Fatal("expected CheckAndUpdateVersion to be called")
	}
	if repo.baseVersion != 0 || repo.newVersion != 1 {
		t.Fatalf("expected expected-version check 0 -> 1, got %d -> %d", repo.baseVersion, repo.newVersion)
	}
}

func TestAggregateStoreSaveRejectsUnexpectedVersion(t *testing.T) {
	repo := &fakeLedgerEventStore{checkOK: false}
	store := &aggregateStoreImpl{repo: repo, outboxRepo: &fakeLedgerOutboxEventsRepo{}}

	aggregate, err := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}
	if _, err := aggregate.BookPayment("payment:pay-1:succeeded", "pay-1", "ledger:clearing:provider:stripe", "VND", 300, time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("BookPayment() error = %v", err)
	}

	err = store.Save(context.Background(), aggregate)
	if err == nil {
		t.Fatal("expected optimistic concurrency error")
	}
	if !strings.Contains(err.Error(), "optimistic concurrency control failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
