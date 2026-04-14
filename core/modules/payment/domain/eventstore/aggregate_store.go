package eventstore

import (
	"context"
	"go-socket/core/shared/pkg/event"

	"gorm.io/gorm"
)

//go:generate mockgen -package=eventstore -destination=aggregate_store_mock.go -source=aggregate_store.go
type AggregateStore interface {
	GetAggregate(ctx context.Context, aggregateID string) (event.Aggregate, error)
	SaveAggregate(ctx context.Context, aggregate event.Aggregate) error
}

type aggregateStore struct {
	db *gorm.DB
}
