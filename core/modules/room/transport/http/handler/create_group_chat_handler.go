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

type createGroupChatHandler struct {
	createGroupChat cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatConversationResponse]
}

func NewCreateGroupChatHandler(
	createGroupChat cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatConversationResponse],
) *createGroupChatHandler {
	return &createGroupChatHandler{
		createGroupChat: createGroupChat,
	}
}

func (h *createGroupChatHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.CreateGroupChatRequest
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

	result, err := h.createGroupChat.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("CreateGroupChat failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
