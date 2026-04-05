// CODE_GENERATOR: handler
package handler

import (
	"errors"
	roomin "go-socket/core/modules/room/application/dto/in"
	roomout "go-socket/core/modules/room/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type deleteChatMessageHandler struct {
	deleteChatMessage cqrs.Dispatcher[*roomin.DeleteChatMessageRequest, *roomout.DeleteChatMessageResponse]
}

func NewDeleteChatMessageHandler(deleteChatMessage cqrs.Dispatcher[*roomin.DeleteChatMessageRequest, *roomout.DeleteChatMessageResponse]) *deleteChatMessageHandler {
	return &deleteChatMessageHandler{
		deleteChatMessage: deleteChatMessage,
	}
}

func (h *deleteChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.DeleteChatMessageRequest{
		MessageID: c.Param("message_id"),
		Scope:     c.Query("scope"),
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.deleteChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("DeleteChatMessage failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("DeleteChatMessage failed"))
	}
	return result, nil
}
