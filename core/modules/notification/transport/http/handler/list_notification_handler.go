package handler

import (
	"errors"
	"go-socket/core/modules/notification/application/dto/in"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type listNotificationHandler struct {
	listNotification cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse]
}

func NewListNotificationHandler(listNotification cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse]) *listNotificationHandler {
	return &listNotificationHandler{listNotification: listNotification}
}

func (h *listNotificationHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.ListNotificationRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.listNotification.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("ListNotification failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("ListNotification failed"))
	}
	return result, nil
}
