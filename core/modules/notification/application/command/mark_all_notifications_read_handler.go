package command

import (
	"context"
	"time"

	"wechat-clone/core/modules/notification/application/dto/in"
	"wechat-clone/core/modules/notification/application/dto/out"
	notificationsupport "wechat-clone/core/modules/notification/application/support"
	notificationservice "wechat-clone/core/modules/notification/application/service"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type markAllNotificationsReadHandler struct {
	baseRepo notificationrepos.Repos
	realtime notificationservice.RealtimeService
}

func NewMarkAllNotificationsReadHandler(
	baseRepo notificationrepos.Repos,
	realtime notificationservice.RealtimeService,
) cqrs.Handler[*in.MarkAllNotificationsReadRequest, *out.MarkAllNotificationsReadResponse] {
	return &markAllNotificationsReadHandler{
		baseRepo: baseRepo,
		realtime: realtime,
	}
}

func (h *markAllNotificationsReadHandler) Handle(ctx context.Context, _ *in.MarkAllNotificationsReadRequest) (*out.MarkAllNotificationsReadResponse, error) {
	log := logging.FromContext(ctx).Named("MarkAllNotificationsRead")
	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		log.Errorw("account not found in context", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	items, err := h.baseRepo.NotificationRepository().ListUnreadByAccountID(ctx, accountID, 0)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	updatedCount := 0
	now := time.Now().UTC()
	for _, item := range items {
		if item == nil {
			continue
		}
		notificationAgg, err := h.baseRepo.NotificationRepository().Load(ctx, item.ID)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		changed, err := notificationAgg.MarkRead(now)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		if !changed {
			continue
		}
		if err := h.baseRepo.NotificationRepository().Save(ctx, notificationAgg); err != nil {
			return nil, stackErr.Error(err)
		}
		updatedCount++
	}

	if h.realtime != nil && updatedCount > 0 {
		if emitErr := h.realtime.EmitMessage(ctx, notificationsupport.NewRealtimeReadAllPayload(accountID)); emitErr != nil {
			log.Errorw("emit notification read-all realtime failed", zap.Error(emitErr))
			return nil, stackErr.Error(emitErr)
		}
	}

	return &out.MarkAllNotificationsReadResponse{
		UpdatedCount: updatedCount,
		UnreadCount:  0,
	}, nil
}
