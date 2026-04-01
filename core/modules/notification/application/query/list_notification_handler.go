package query

import (
	"context"
	"errors"
	"go-socket/core/modules/notification/application/dto/in"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/domain/repos"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"

	"go.uber.org/zap"
)

type listNotificationHandler struct {
	notificationRepo repos.NotificationRepository
}

func NewListNotificationHandler(repos repos.Repos) cqrs.Handler[*in.ListNotificationRequest, *out.ListNotificationResponse] {
	return &listNotificationHandler{
		notificationRepo: repos.NotificationRepository(),
	}
}

func (h *listNotificationHandler) Handle(ctx context.Context, req *in.ListNotificationRequest) (*out.ListNotificationResponse, error) {
	log := logging.FromContext(ctx).Named("ListNotification")
	account := ctx.Value("account")
	if account == nil {
		log.Errorw("Account not found", zap.Error(errors.New("account not found")))
		return nil, stackerr.Error(errors.New("account not found"))
	}
	payload, ok := account.(*xpaseto.PasetoPayload)
	if !ok {
		return nil, stackerr.Error(errors.New("invalid account payload"))
	}
	options := utils.QueryOptions{
		Conditions: []utils.Condition{
			{
				Field:    "account_id",
				Value:    payload.AccountID,
				Operator: utils.Equal,
			},
		},
	}
	if req.Cursor != "" {
		createdAt, id, err := utils.DecodeCursor(req.Cursor)
		if err != nil {
			log.Errorw("Invalid cursor", zap.Error(err))
			return nil, stackerr.Error(err)
		}
		options.Conditions = append(options.Conditions, utils.Condition{
			Field:    "(created_at < ? OR (created_at = ? AND id < ?))",
			Operator: utils.Raw,
			Value:    []interface{}{createdAt, createdAt, id},
		})
	}
	limit := req.Limit
	if limit > 0 {
		queryLimit := limit + 1
		options.Limit = &queryLimit
	}
	notifications, err := h.notificationRepo.ListNotifications(ctx, options)
	if err != nil {
		log.Errorw("Failed to list notifications", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	hasMore := false
	if limit > 0 && len(notifications) > limit {
		hasMore = true
		notifications = notifications[:limit]
	}

	items := make([]out.NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		items = append(items, *notification)
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = utils.EncodeCursor(last.CreatedAt, last.ID)
	}

	return &out.ListNotificationResponse{
		Notifications: items,
		NextCursor:    nextCursor,
		HasMore:       hasMore,
		Limit:         req.Limit,
		Total:         len(items),
	}, nil
}
