package eventstore

import (
	"context"
	eventpkg "wechat-clone/core/shared/pkg/event"
)

//go:generate mockgen -package=eventstore -destination=aggregate_event_store_mock.go -source=aggregate_event_store.go
type LedgerEventStore interface {
	CreateIfNotExist(ctx context.Context, aggregateID, aggregateType string) error
	CheckAndUpdateVersion(ctx context.Context, aggregateID, aggregateType string, baseVersion, newVersion int) (bool, error)
	Append(ctx context.Context, evt eventpkg.Event) error
	Get(ctx context.Context, aggregateID, aggregateType string, afterVersion int, agg eventpkg.Aggregate) error
	CreateSnapshot(ctx context.Context, agg eventpkg.Aggregate) error
	ReadSnapshot(ctx context.Context, aggregateID, aggregateType string, agg eventpkg.Aggregate) (bool, error)
}

// LedgerPostingStore persists the booking uniqueness/index derived from ledger events.
// It is not part of the aggregate event stream; it exists to enforce idempotent posting writes.
//
//go:generate mockgen -package=eventstore -destination=aggregate_event_store_mock.go -source=aggregate_event_store.go
type LedgerPostingStore interface {
	ReservePostedTransaction(ctx context.Context, evt eventpkg.Event) error
}
