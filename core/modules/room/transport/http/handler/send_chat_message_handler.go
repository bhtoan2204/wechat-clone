// CODE_GENERATOR - do not edit: handler
package handler

import (
	"net/http"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type sendChatMessageHandler struct {
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageResponse]
}

func NewSendChatMessageHandler(
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageResponse],
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
