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

type createDirectConversationHandler struct {
	createDirectConversation cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatRoomCommandResponse]
}

func NewCreateDirectConversationHandler(
	createDirectConversation cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatRoomCommandResponse],
) *createDirectConversationHandler {
	return &createDirectConversationHandler{
		createDirectConversation: createDirectConversation,
	}
}

func (h *createDirectConversationHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.CreateDirectConversationRequest
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

	result, err := h.createDirectConversation.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("CreateDirectConversation failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
