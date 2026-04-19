package command

import (
	"context"
	"fmt"
	"time"

	"wechat-clone/core/modules/notification/application/dto/in"
	"wechat-clone/core/modules/notification/application/dto/out"
	notificationsupport "wechat-clone/core/modules/notification/application/support"
	notificationservice "wechat-clone/core/modules/notification/application/service"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	notificationtypes "wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type markNotificationReadHandler struct {
	baseRepo  notificationrepos.Repos
	realtime  notificationservice.RealtimeService
}

func NewMarkNotificationReadHandler(
	baseRepo notificationrepos.Repos,
	realtime notificationservice.RealtimeService,
) cqrs.Handler[*in.MarkNotificationReadRequest, *out.MarkNotificationReadResponse] {
	return &markNotificationReadHandler{
		baseRepo: baseRepo,
		realtime: realtime,
	}
}

func (h *markNotificationReadHandler) Handle(ctx context.Context, req *in.MarkNotificationReadRequest) (*out.MarkNotificationReadResponse, error) {
	log := logging.FromContext(ctx).Named("MarkNotificationRead")
	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		log.Errorw("account not found in context", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	notificationAgg, err := h.baseRepo.NotificationRepository().Load(ctx, req.NotificationID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	snapshot, err := notificationAgg.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if snapshot.AccountID != accountID {
		return nil, stackErr.Error(fmt.Errorf("notification does not belong to account"))
	}

	changed, err := notificationAgg.MarkRead(time.Now().UTC())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if changed {
		if err := h.baseRepo.NotificationRepository().Save(ctx, notificationAgg); err != nil {
			return nil, stackErr.Error(err)
		}
	}

	current, err := notificationAgg.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	unreadCount, err := h.baseRepo.NotificationRepository().CountUnread(ctx, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if changed && h.realtime != nil {
		payload := notificationsupport.NewRealtimeNotificationPayload(notificationtypes.RealtimeEventNotificationRead, current, unreadCount)
		if emitErr := h.realtime.EmitMessage(ctx, payload); emitErr != nil {
			log.Errorw("emit notification read realtime failed", zap.Error(emitErr))
			return nil, stackErr.Error(emitErr)
		}
	}

	return &out.MarkNotificationReadResponse{
		Notification: func() *out.NotificationResponse {
			response := notificationsupport.ToNotificationResponse(current)
			return &response
		}(),
		UnreadCount:  unreadCount,
	}, nil
}
