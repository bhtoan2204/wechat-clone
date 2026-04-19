package query

import (
	"context"

	"wechat-clone/core/modules/notification/domain/entity"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
)

//go:generate mockgen -package=query -destination=notification_read_repository_mock.go -source=notification_read_repository.go
type NotificationReadRepository interface {
	ListByAccountID(ctx context.Context, accountID string, cursor *notificationrepos.NotificationListCursor, limit int) ([]*entity.NotificationEntity, error)
	CountUnread(ctx context.Context, accountID string) (int, error)
}
