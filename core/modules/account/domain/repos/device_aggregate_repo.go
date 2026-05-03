package repos

import (
	"context"

	"wechat-clone/core/modules/account/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=device_aggregate_repo_mock.go -source=device_aggregate_repo.go
type DeviceAggregateRepository interface {
	GetByAccountAndID(ctx context.Context, accountID string, deviceID string) (*aggregate.DeviceAggregate, error)
	Save(ctx context.Context, device *aggregate.DeviceAggregate) error
}

type DeviceRepository = DeviceAggregateRepository
