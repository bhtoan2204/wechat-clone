package repos

import (
	"context"

	"go-socket/core/modules/room/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=room_aggregate_repo_mock.go -source=room_aggregate_repo.go
type RoomAggregateRepository interface {
	Load(ctx context.Context, roomID string) (*aggregate.RoomStateAggregate, error)
	LoadByDirectKey(ctx context.Context, directKey string) (*aggregate.RoomStateAggregate, error)
	Save(ctx context.Context, agg *aggregate.RoomStateAggregate) error
	Delete(ctx context.Context, roomID string) error
}
