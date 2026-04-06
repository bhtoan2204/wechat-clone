// CODE_GENERATOR: handler
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

type getChatConversationHandler struct {
	getChatConversation cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse]
}

func NewGetChatConversationHandler(
	getChatConversation cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse],
) *getChatConversationHandler {
	return &getChatConversationHandler{
		getChatConversation: getChatConversation,
	}
}

func (h *getChatConversationHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.GetChatConversationRequest
	request.RoomID = c.Param("room_id")

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	result, err := h.getChatConversation.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetChatConversation failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
