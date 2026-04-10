package query

import (
	"context"

	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/shared/utils"
)

type NotificationReadRepository interface {
	ListNotifications(ctx context.Context, options utils.QueryOptions) ([]*out.NotificationResponse, error)
}
