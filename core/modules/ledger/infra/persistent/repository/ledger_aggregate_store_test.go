package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	"wechat-clone/core/modules/ledger/domain/eventstore"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"

	"go.uber.org/mock/gomock"
)

func TestAggregateStoreSaveUsesExpectedVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := eventstore.NewMockLedgerEventStore(ctrl)
	postingStore := eventstore.NewMockLedgerPostingStore(ctrl)
	outboxRepo := ledgerrepos.NewMockLedgerOutboxEventsRepository(ctrl)

	store := &aggregateStoreImpl{
		repo:         repo,
		postingStore: postingStore,
		outboxRepo:   outboxRepo,
	}

	aggregate, err := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}
	if _, err := aggregate.BookPayment(
		"payment:pay-1:succeeded",
		"pay-1",
		"ledger:clearing:provider:stripe",
		"VND",
		300,
		time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("BookPayment() error = %v", err)
	}

	repo.EXPECT().
		CreateIfNotExist(gomock.Any(), "wallet:available", gomock.Any()).
		Return(nil)

	postingStore.EXPECT().
		ReservePostedTransaction(gomock.Any(), gomock.Any()).
		Return(nil)

	repo.EXPECT().
		CheckAndUpdateVersion(gomock.Any(), "wallet:available", gomock.Any(), 0, 1).
		Return(true, nil)

	repo.EXPECT().
		Append(gomock.Any(), gomock.Any()).
		Return(nil)

	outboxRepo.EXPECT().
		Append(gomock.Any(), gomock.Any()).
		Return(nil)

	repo.EXPECT().
		CreateSnapshot(gomock.Any(), gomock.Any()).
		Times(0)

	if err := store.Save(context.Background(), aggregate); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestAggregateStoreSaveRejectsUnexpectedVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := eventstore.NewMockLedgerEventStore(ctrl)
	postingStore := eventstore.NewMockLedgerPostingStore(ctrl)
	outboxRepo := ledgerrepos.NewMockLedgerOutboxEventsRepository(ctrl)

	store := &aggregateStoreImpl{
		repo:         repo,
		postingStore: postingStore,
		outboxRepo:   outboxRepo,
	}

	aggregate, err := ledgeraggregate.NewLedgerAccountAggregate("wallet:available")
	if err != nil {
		t.Fatalf("NewLedgerAccountAggregate() error = %v", err)
	}
	if _, err := aggregate.BookPayment(
		"payment:pay-1:succeeded",
		"pay-1",
		"ledger:clearing:provider:stripe",
		"VND",
		300,
		time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("BookPayment() error = %v", err)
	}

	repo.EXPECT().
		CreateIfNotExist(gomock.Any(), "wallet:available", gomock.Any()).
		Return(nil)

	postingStore.EXPECT().
		ReservePostedTransaction(gomock.Any(), gomock.Any()).
		Return(nil)

	repo.EXPECT().
		CheckAndUpdateVersion(gomock.Any(), "wallet:available", gomock.Any(), 0, 1).
		Return(false, nil)

	err = store.Save(context.Background(), aggregate)
	if err == nil {
		t.Fatal("expected optimistic concurrency error")
	}
	if !strings.Contains(err.Error(), "optimistic concurrency control failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
