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

type getUnreadNotificationCountHandler struct {
	dispatcher cqrs.Dispatcher[*in.GetUnreadNotificationCountRequest, *out.GetUnreadNotificationCountResponse]
}

func NewGetUnreadNotificationCountHandler(dispatcher cqrs.Dispatcher[*in.GetUnreadNotificationCountRequest, *out.GetUnreadNotificationCountResponse]) *getUnreadNotificationCountHandler {
	return &getUnreadNotificationCountHandler{dispatcher: dispatcher}
}

func (h *getUnreadNotificationCountHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)

	result, err := h.dispatcher.Dispatch(ctx, &in.GetUnreadNotificationCountRequest{})
	if err != nil {
		logger.Errorw("get unread notification count failed", zap.Error(err))
		return nil, stackErr.Error(errors.New("get unread notification count failed"))
	}
	return result, nil
}
