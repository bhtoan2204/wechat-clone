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

type updateGroupChatHandler struct {
	updateGroupChat cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatConversationResponse]
}

func NewUpdateGroupChatHandler(
	updateGroupChat cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatConversationResponse],
) *updateGroupChatHandler {
	return &updateGroupChatHandler{
		updateGroupChat: updateGroupChat,
	}
}

func (h *updateGroupChatHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.UpdateGroupChatRequest
	request.RoomID = c.Param("room_id")
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	result, err := h.updateGroupChat.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("UpdateGroupChat failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
