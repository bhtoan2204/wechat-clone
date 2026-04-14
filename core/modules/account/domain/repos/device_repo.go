package repos

import (
	"context"

	"go-socket/core/modules/account/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=device_repo_mock.go -source=device_repo.go
type DeviceRepository interface {
	FindByAccountAndUID(ctx context.Context, accountID string, deviceUID string) (*aggregate.DeviceAggregate, error)
	GetByAccountAndID(ctx context.Context, accountID string, deviceID string) (*aggregate.DeviceAggregate, error)
	Save(ctx context.Context, device *aggregate.DeviceAggregate) error
}
