package eventstore

import (
	"context"
	eventpkg "wechat-clone/core/shared/pkg/event"
)

//go:generate mockgen -package=eventstore -destination=aggregate_store_mock.go -source=aggregate_store.go
type AggregateStore interface {
	Get(ctx context.Context, aggregateID string, agg eventpkg.Aggregate) error
	Save(ctx context.Context, agg eventpkg.Aggregate) error
}
