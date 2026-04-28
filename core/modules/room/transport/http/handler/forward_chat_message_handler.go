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

type forwardChatMessageHandler struct {
	forwardChatMessage cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageCommandResponse]
}

func NewForwardChatMessageHandler(
	forwardChatMessage cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageCommandResponse],
) *forwardChatMessageHandler {
	return &forwardChatMessageHandler{
		forwardChatMessage: forwardChatMessage,
	}
}

func (h *forwardChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.ForwardChatMessageRequest
	request.MessageID = c.Param("message_id")
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

	result, err := h.forwardChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("ForwardChatMessage failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
