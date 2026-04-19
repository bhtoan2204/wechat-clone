package handler

import (
	"errors"

	"wechat-clone/core/modules/notification/application/dto/in"
	"wechat-clone/core/modules/notification/application/dto/out"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type markNotificationReadHandler struct {
	dispatcher cqrs.Dispatcher[*in.MarkNotificationReadRequest, *out.MarkNotificationReadResponse]
}

func NewMarkNotificationReadHandler(dispatcher cqrs.Dispatcher[*in.MarkNotificationReadRequest, *out.MarkNotificationReadResponse]) *markNotificationReadHandler {
	return &markNotificationReadHandler{dispatcher: dispatcher}
}

func (h *markNotificationReadHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)

	request := in.MarkNotificationReadRequest{
		NotificationID: c.Param("notification_id"),
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("validate request failed", zap.Error(err))
		return nil, stackErr.Error(errors.New("validate request failed"))
	}

	result, err := h.dispatcher.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("mark notification read failed", zap.Error(err))
		return nil, stackErr.Error(errors.New("mark notification read failed"))
	}
	return result, nil
}
