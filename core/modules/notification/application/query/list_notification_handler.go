package query

import (
	"context"
	"time"

	"wechat-clone/core/modules/notification/application/dto/in"
	"wechat-clone/core/modules/notification/application/dto/out"
	notificationsupport "wechat-clone/core/modules/notification/application/support"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"go.uber.org/zap"
)

type listNotificationHandler struct {
	notificationRepo NotificationReadRepository
}

func NewListNotificationHandler(notificationRepo NotificationReadRepository) cqrs.Handler[*in.ListNotificationRequest, *out.ListNotificationResponse] {
	return &listNotificationHandler{
		notificationRepo: notificationRepo,
	}
}

func (h *listNotificationHandler) Handle(ctx context.Context, req *in.ListNotificationRequest) (*out.ListNotificationResponse, error) {
	log := logging.FromContext(ctx).Named("ListNotification")
	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		log.Errorw("account not found in context", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	var cursor *notificationrepos.NotificationListCursor
	if req.Cursor != "" {
		sortAt, notificationID, err := utils.DecodeCursor(req.Cursor)
		if err != nil {
			log.Errorw("decode notification cursor failed", zap.Error(err))
			return nil, stackErr.Error(err)
		}
		cursor = &notificationrepos.NotificationListCursor{
			SortAt:         sortAt.UTC(),
			NotificationID: notificationID,
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	items, err := h.notificationRepo.ListByAccountID(ctx, accountID, cursor, limit+1)
	if err != nil {
		log.Errorw("list notifications failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	unreadCount, err := h.notificationRepo.CountUnread(ctx, accountID)
	if err != nil {
		log.Errorw("count unread notifications failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	hasMore := false
	if len(items) > limit {
		hasMore = true
		items = items[:limit]
	}

	responses := make([]out.NotificationResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, notificationsupport.ToNotificationResponse(item))
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = utils.EncodeCursor(last.SortAt.UTC().Format(time.RFC3339Nano), last.ID)
	}

	return &out.ListNotificationResponse{
		Notifications: responses,
		NextCursor:    nextCursor,
		HasMore:       hasMore,
		Total:         len(responses),
		Limit:         limit,
		UnreadCount:   unreadCount,
	}, nil
}
