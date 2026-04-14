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

type deleteChatMessageHandler struct {
	deleteChatMessage cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse]
}

func NewDeleteChatMessageHandler(
	deleteChatMessage cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse],
) *deleteChatMessageHandler {
	return &deleteChatMessageHandler{
		deleteChatMessage: deleteChatMessage,
	}
}

func (h *deleteChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.DeleteChatMessageRequest
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

	result, err := h.deleteChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("DeleteChatMessage failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
