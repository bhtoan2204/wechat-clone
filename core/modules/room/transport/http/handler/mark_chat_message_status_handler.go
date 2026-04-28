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

type markChatMessageStatusHandler struct {
	markChatMessageStatus cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.ChatMessageCommandResponse]
}

func NewMarkChatMessageStatusHandler(
	markChatMessageStatus cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.ChatMessageCommandResponse],
) *markChatMessageStatusHandler {
	return &markChatMessageStatusHandler{
		markChatMessageStatus: markChatMessageStatus,
	}
}

func (h *markChatMessageStatusHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.MarkChatMessageStatusRequest
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

	result, err := h.markChatMessageStatus.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("MarkChatMessageStatus failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
