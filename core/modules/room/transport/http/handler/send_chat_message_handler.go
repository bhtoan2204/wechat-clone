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

type sendChatMessageHandler struct {
	sendChatMessage cqrs.Dispatcher[*roomin.SendChatMessageRequest, *roomout.ChatMessageResponse]
}

func NewSendChatMessageHandler(sendChatMessage cqrs.Dispatcher[*roomin.SendChatMessageRequest, *roomout.ChatMessageResponse]) *sendChatMessageHandler {
	return &sendChatMessageHandler{
		sendChatMessage: sendChatMessage,
	}
}

func (h *sendChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request roomin.SendChatMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.sendChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("SendChatMessage failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("SendChatMessage failed"))
	}
	return result, nil
}
