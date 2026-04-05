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

type forwardChatMessageHandler struct {
	forwardChatMessage cqrs.Dispatcher[*roomin.ForwardChatMessageRequest, *roomout.ChatMessageResponse]
}

func NewForwardChatMessageHandler(forwardChatMessage cqrs.Dispatcher[*roomin.ForwardChatMessageRequest, *roomout.ChatMessageResponse]) *forwardChatMessageHandler {
	return &forwardChatMessageHandler{
		forwardChatMessage: forwardChatMessage,
	}
}

func (h *forwardChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.ForwardChatMessageRequest{MessageID: c.Param("message_id")}
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.forwardChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("ForwardChatMessage failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("ForwardChatMessage failed"))
	}
	return result, nil
}
