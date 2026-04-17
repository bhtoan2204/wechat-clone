package repository

import (
	"context"
	"fmt"
	"reflect"

	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

type aggregateStore interface {
	Get(ctx context.Context, aggregateID string, agg eventpkg.Aggregate) error
	Save(ctx context.Context, agg eventpkg.Aggregate) error
}

type aggregateStoreImpl struct {
	repo ledgerEventStore
}

func newAggregateStore(dbTX dbTX, serializer eventpkg.Serializer) aggregateStore {
	return &aggregateStoreImpl{
		repo: newLedgerEventStore(dbTX, serializer),
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
			return stackErr.Error(fmt.Errorf("append ledger event #%d failed: %v", idx, err))
		}
		if evt.Version%10 == 0 {
			if err := s.repo.CreateSnapshot(ctx, agg); err != nil {
				return stackErr.Error(fmt.Errorf("create ledger snapshot failed: %v", err))
			}
		}
	}

	root.Update()
	return nil
}
