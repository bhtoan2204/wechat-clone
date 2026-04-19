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

type markAllNotificationsReadHandler struct {
	dispatcher cqrs.Dispatcher[*in.MarkAllNotificationsReadRequest, *out.MarkAllNotificationsReadResponse]
}

func NewMarkAllNotificationsReadHandler(dispatcher cqrs.Dispatcher[*in.MarkAllNotificationsReadRequest, *out.MarkAllNotificationsReadResponse]) *markAllNotificationsReadHandler {
	return &markAllNotificationsReadHandler{dispatcher: dispatcher}
}

func (h *markAllNotificationsReadHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)

	result, err := h.dispatcher.Dispatch(ctx, &in.MarkAllNotificationsReadRequest{})
	if err != nil {
		logger.Errorw("mark all notifications read failed", zap.Error(err))
		return nil, stackErr.Error(errors.New("mark all notifications read failed"))
	}
	return result, nil
}
