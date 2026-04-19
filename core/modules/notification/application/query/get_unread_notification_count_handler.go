package query

import (
	"context"

	"wechat-clone/core/modules/notification/application/dto/in"
	"wechat-clone/core/modules/notification/application/dto/out"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type getUnreadNotificationCountHandler struct {
	notificationRepo NotificationReadRepository
}

func NewGetUnreadNotificationCountHandler(notificationRepo NotificationReadRepository) cqrs.Handler[*in.GetUnreadNotificationCountRequest, *out.GetUnreadNotificationCountResponse] {
	return &getUnreadNotificationCountHandler{notificationRepo: notificationRepo}
}

func (h *getUnreadNotificationCountHandler) Handle(ctx context.Context, _ *in.GetUnreadNotificationCountRequest) (*out.GetUnreadNotificationCountResponse, error) {
	log := logging.FromContext(ctx).Named("GetUnreadNotificationCount")
	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		log.Errorw("account not found in context", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	unreadCount, err := h.notificationRepo.CountUnread(ctx, accountID)
	if err != nil {
		log.Errorw("count unread notifications failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	return &out.GetUnreadNotificationCountResponse{UnreadCount: unreadCount}, nil
}
