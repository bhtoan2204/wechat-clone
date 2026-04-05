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

type pinChatMessageHandler struct {
	pinChatMessage cqrs.Dispatcher[*roomin.PinChatMessageRequest, *roomout.ChatConversationResponse]
}

func NewPinChatMessageHandler(pinChatMessage cqrs.Dispatcher[*roomin.PinChatMessageRequest, *roomout.ChatConversationResponse]) *pinChatMessageHandler {
	return &pinChatMessageHandler{
		pinChatMessage: pinChatMessage,
	}
}

func (h *pinChatMessageHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.PinChatMessageRequest{RoomID: c.Param("room_id")}
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.pinChatMessage.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("PinChatMessage failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("PinChatMessage failed"))
	}
	return result, nil
}
