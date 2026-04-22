package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	appprojection "wechat-clone/core/modules/ledger/application/projection"
	"wechat-clone/core/modules/ledger/domain/eventstore"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

type aggregateStoreImpl struct {
	repo         eventstore.LedgerEventStore
	postingStore eventstore.LedgerPostingStore
	outboxRepo   ledgerrepos.LedgerOutboxEventsRepository
}

func newAggregateStore(dbTX dbTX, serializer eventpkg.Serializer) eventstore.AggregateStore {
	return &aggregateStoreImpl{
		repo:         newLedgerEventStore(dbTX, serializer),
		postingStore: newLedgerPostedTransactionStore(dbTX, serializer),
		outboxRepo:   NewLedgerOutboxEventsRepoImpl(dbTX),
	}
}

func (s *aggregateStoreImpl) Get(ctx context.Context, aggregateID string, agg eventpkg.Aggregate) error {
	if reflect.ValueOf(agg).Kind() != reflect.Ptr {
		return stackErr.Error(fmt.Errorf("aggregate must be a pointer"))
	}

	aggregateType := eventpkg.AggregateTypeName(agg)
	agg.Root().SetAggregateType(aggregateType)

	hasSnapshot, err := s.repo.ReadSnapshot(ctx, aggregateID, aggregateType, agg)
	if err != nil {
		return stackErr.Error(err)
	}
	if !hasSnapshot {
		return stackErr.Error(s.repo.Get(ctx, aggregateID, aggregateType, 0, agg))
	}

	return stackErr.Error(s.repo.Get(ctx, aggregateID, aggregateType, agg.Root().BaseVersion(), agg))
}

func (s *aggregateStoreImpl) Save(ctx context.Context, agg eventpkg.Aggregate) error {
	if reflect.ValueOf(agg).Kind() != reflect.Ptr {
		return stackErr.Error(fmt.Errorf("aggregate must be a pointer"))
	}

	root := agg.Root()
	aggregateType := eventpkg.AggregateTypeName(agg)
	root.SetAggregateType(aggregateType)

	events := root.CloneEvents()
	if len(events) == 0 {
		return nil
	}

	if err := s.repo.CreateIfNotExist(ctx, root.AggregateID(), aggregateType); err != nil {
		return stackErr.Error(err)
	}

	for idx, evt := range events {
		if err := s.postingStore.ReservePostedTransaction(ctx, evt); err != nil {
			if errors.Is(err, ledgerrepos.ErrAlreadyApplied) {
				if len(events) == 1 && idx == 0 {
					return stackErr.Error(err)
				}
				return stackErr.Error(fmt.Errorf("ledger idempotency collision on event #%d: %w", idx, err))
			}
			return stackErr.Error(fmt.Errorf("reserve ledger posting #%d failed: %w", idx, err))
		}
	}

	if ok, err := s.repo.CheckAndUpdateVersion(ctx, root.AggregateID(), aggregateType, root.BaseVersion(), root.Version()); err != nil {
		return stackErr.Error(err)
	} else if !ok {
		return stackErr.Error(fmt.Errorf(
			"optimistic concurrency control failed id=%s expectedVersion=%d newVersion=%d",
			root.AggregateID(),
			root.BaseVersion(),
			root.Version(),
		))
	}

	for idx, evt := range events {
		if err := s.repo.Append(ctx, evt); err != nil {
			return stackErr.Error(fmt.Errorf("append ledger event #%d failed: %w", idx, err))
		}
		// Ledger events are the aggregate history. Projection consumption gets the same
		// internal event copied to outbox so rebuild and downstream projection stay aligned.
		if projectionEvent, ok := ledgerProjectionOutboxEvent(evt); ok && s.outboxRepo != nil {
			if err := s.outboxRepo.Append(ctx, projectionEvent); err != nil {
				return stackErr.Error(fmt.Errorf("append ledger projection outbox event #%d failed: %w", idx, err))
			}
		}
		if evt.Version%100 == 0 {
			if err := s.repo.CreateSnapshot(ctx, agg); err != nil {
				return stackErr.Error(fmt.Errorf("create ledger snapshot failed: %w", err))
			}
		}
	}

	root.Update()
	return nil
}

func ledgerProjectionOutboxEvent(evt eventpkg.Event) (eventpkg.Event, bool) {
	if !appprojection.IsLedgerTransactionProjectionEvent(evt.EventName) {
		return eventpkg.Event{}, false
	}
	return evt, true
}
