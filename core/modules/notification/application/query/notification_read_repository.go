package query

import (
	"context"

	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/shared/utils"
)

//go:generate mockgen -package=query -destination=notification_read_repository_mock.go -source=notification_read_repository.go
type NotificationReadRepository interface {
	ListNotifications(ctx context.Context, options utils.QueryOptions) ([]*out.NotificationResponse, error)
}
