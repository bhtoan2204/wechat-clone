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

type getChatConversationMetadataHandler struct {
	getChatConversationMetadata cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationMetadataResponse]
}

func NewGetChatConversationMetadataHandler(
	getChatConversationMetadata cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationMetadataResponse],
) *getChatConversationMetadataHandler {
	return &getChatConversationMetadataHandler{getChatConversationMetadata: getChatConversationMetadata}
}

func (h *getChatConversationMetadataHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.GetChatConversationRequest
	request.RoomID = c.Param("room_id")

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.getChatConversationMetadata.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetChatConversationMetadata failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
