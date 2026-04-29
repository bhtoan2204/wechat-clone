package repos

import (
	"context"

	"wechat-clone/core/modules/room/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=room_aggregate_repo_mock.go -source=room_aggregate_repo.go
type RoomAggregateRepository interface {
	Load(ctx context.Context, roomID string) (*aggregate.RoomAggregate, error)
	LoadByDirectKey(ctx context.Context, directKey string) (*aggregate.RoomAggregate, error)
	Save(ctx context.Context, agg *aggregate.RoomAggregate) error
	Delete(ctx context.Context, roomID string) error
}
