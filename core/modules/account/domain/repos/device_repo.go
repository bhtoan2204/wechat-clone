package repos

import (
	"context"

	"go-socket/core/modules/account/domain/entity"
)

//go:generate mockgen -package=repos -destination=device_repo_mock.go -source=device_repo.go
type DeviceRepository interface {
	FindByAccountAndUID(ctx context.Context, accountID string, deviceUID string) (*entity.Device, error)
	GetByAccountAndID(ctx context.Context, accountID string, deviceID string) (*entity.Device, error)
	Save(ctx context.Context, device *entity.Device) error
}
