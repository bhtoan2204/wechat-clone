package repos

import (
	"context"
	"time"

	"wechat-clone/core/modules/notification/domain/aggregate"
	"wechat-clone/core/modules/notification/domain/entity"
)

type NotificationListCursor struct {
	SortAt         time.Time
	NotificationID string
}

//go:generate mockgen -package=repos -destination=notification_repo_mock.go -source=notification_repo.go
type NotificationRepository interface {
	Load(ctx context.Context, notificationID string) (*aggregate.NotificationAggregate, error)
	LoadMessageGroup(ctx context.Context, accountID, groupKey string) (*aggregate.NotificationAggregate, error)
	Save(ctx context.Context, notification *aggregate.NotificationAggregate) error
	ListByAccountID(ctx context.Context, accountID string, cursor *NotificationListCursor, limit int) ([]*entity.NotificationEntity, error)
	ListUnreadByAccountID(ctx context.Context, accountID string, limit int) ([]*entity.NotificationEntity, error)
	CountUnread(ctx context.Context, accountID string) (int, error)
}
