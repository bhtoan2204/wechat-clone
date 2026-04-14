package repos

import (
	"context"
	"go-socket/core/modules/notification/domain/entity"
)

//go:generate mockgen -package=repos -destination=notification_repo_mock.go -source=notification_repo.go
type NotificationRepository interface {
	CreateNotification(ctx context.Context, notification *entity.NotificationEntity) error
}
