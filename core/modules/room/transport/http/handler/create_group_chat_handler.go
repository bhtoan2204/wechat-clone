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

type createGroupChatHandler struct {
	createGroupChat cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatRoomCommandResponse]
}

func NewCreateGroupChatHandler(
	createGroupChat cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatRoomCommandResponse],
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
