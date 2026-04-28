// CODE_GENERATOR - do not edit: handler
package handler

import (
	"net/http"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type sendChatMessageHandler struct {
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageCommandResponse]
}

func NewSendChatMessageHandler(
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageCommandResponse],
) *sendChatMessageHandler {
	return &sendChatMessageHandler{
		sendChatMessage: sendChatMessage,
	}
}

func (h *sendChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.SendChatMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.sendChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("SendChatMessage failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
