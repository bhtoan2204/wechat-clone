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

type updateGroupChatHandler struct {
	updateGroupChat cqrs.Dispatcher[*roomin.UpdateGroupChatRequest, *roomout.ChatConversationResponse]
}

func NewUpdateGroupChatHandler(updateGroupChat cqrs.Dispatcher[*roomin.UpdateGroupChatRequest, *roomout.ChatConversationResponse]) *updateGroupChatHandler {
	return &updateGroupChatHandler{
		updateGroupChat: updateGroupChat,
	}
}

func (h *updateGroupChatHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.UpdateGroupChatRequest{RoomID: c.Param("room_id")}
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.updateGroupChat.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("UpdateGroupChat failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("UpdateGroupChat failed"))
	}
	return result, nil
}
