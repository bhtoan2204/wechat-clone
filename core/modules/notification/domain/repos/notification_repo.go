package repos

import (
	"context"
	"go-socket/core/modules/notification/domain/entity"
)

type NotificationRepository interface {
	CreateNotification(ctx context.Context, notification *entity.NotificationEntity) error
}
