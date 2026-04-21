package eventstore

import (
	"context"
	eventpkg "wechat-clone/core/shared/pkg/event"
)

type LedgerEventStore interface {
	CreateIfNotExist(ctx context.Context, aggregateID, aggregateType string) error
	CheckAndUpdateVersion(ctx context.Context, aggregateID, aggregateType string, baseVersion, newVersion int) (bool, error)
	ReservePostedTransaction(ctx context.Context, evt eventpkg.Event) error
	Append(ctx context.Context, evt eventpkg.Event) error
	Get(ctx context.Context, aggregateID, aggregateType string, afterVersion int, agg eventpkg.Aggregate) error
	CreateSnapshot(ctx context.Context, agg eventpkg.Aggregate) error
	ReadSnapshot(ctx context.Context, aggregateID, aggregateType string, agg eventpkg.Aggregate) (bool, error)
}
