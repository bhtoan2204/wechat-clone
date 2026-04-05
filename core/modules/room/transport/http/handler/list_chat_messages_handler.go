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

type listChatMessagesHandler struct {
	listChatMessages cqrs.Dispatcher[*roomin.ListChatMessagesRequest, []*roomout.ChatMessageResponse]
}

func NewListChatMessagesHandler(listChatMessages cqrs.Dispatcher[*roomin.ListChatMessagesRequest, []*roomout.ChatMessageResponse]) *listChatMessagesHandler {
	return &listChatMessagesHandler{
		listChatMessages: listChatMessages,
	}
}

func (h *listChatMessagesHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.ListChatMessagesRequest{RoomID: c.Param("room_id")}
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.listChatMessages.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("ListChatMessages failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("ListChatMessages failed"))
	}
	return result, nil
}
