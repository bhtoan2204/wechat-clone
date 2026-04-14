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

type pinChatMessageHandler struct {
	pinChatMessage cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatConversationResponse]
}

func NewPinChatMessageHandler(
	pinChatMessage cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatConversationResponse],
) *pinChatMessageHandler {
	return &pinChatMessageHandler{
		pinChatMessage: pinChatMessage,
	}
}

func (h *pinChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.PinChatMessageRequest
	request.RoomID = c.Param("room_id")
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

	result, err := h.pinChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("PinChatMessage failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
