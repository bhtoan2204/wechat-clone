package eventstore

import (
	"context"
	eventpkg "wechat-clone/core/shared/pkg/event"
)

type AggregateStore interface {
	Get(ctx context.Context, aggregateID string, agg eventpkg.Aggregate) error
	Save(ctx context.Context, agg eventpkg.Aggregate) error
}
