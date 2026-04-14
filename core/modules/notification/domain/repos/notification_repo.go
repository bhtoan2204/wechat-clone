package repos

import (
	"context"
	"go-socket/core/modules/notification/domain/aggregate"
	"go-socket/core/modules/notification/domain/entity"
)

//go:generate mockgen -package=repos -destination=notification_repo_mock.go -source=notification_repo.go
type NotificationRepository interface {
	Load(ctx context.Context, notificationID string) (*aggregate.NotificationAggregate, error)
	Save(ctx context.Context, notification *aggregate.NotificationAggregate) error
	ListByAccountID(ctx context.Context, accountID string) ([]*entity.NotificationEntity, error)
}
