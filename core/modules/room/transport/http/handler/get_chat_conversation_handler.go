// CODE_GENERATOR: handler
package handler

import (
	"errors"
	roomin "go-socket/core/modules/room/application/dto/in"
	roomout "go-socket/core/modules/room/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type getChatConversationHandler struct {
	getChatConversation cqrs.Dispatcher[*roomin.GetChatConversationRequest, *roomout.ChatConversationResponse]
}

func NewGetChatConversationHandler(getChatConversation cqrs.Dispatcher[*roomin.GetChatConversationRequest, *roomout.ChatConversationResponse]) *getChatConversationHandler {
	return &getChatConversationHandler{
		getChatConversation: getChatConversation,
	}
}

func (h *getChatConversationHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.GetChatConversationRequest{RoomID: c.Param("room_id")}
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.getChatConversation.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetChatConversation failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("GetChatConversation failed"))
	}
	return result, nil
}
