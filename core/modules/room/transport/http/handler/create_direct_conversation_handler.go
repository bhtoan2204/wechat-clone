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

type createDirectConversationHandler struct {
	createDirectConversation cqrs.Dispatcher[*roomin.CreateDirectConversationRequest, *roomout.ChatConversationResponse]
}

func NewCreateDirectConversationHandler(createDirectConversation cqrs.Dispatcher[*roomin.CreateDirectConversationRequest, *roomout.ChatConversationResponse]) *createDirectConversationHandler {
	return &createDirectConversationHandler{
		createDirectConversation: createDirectConversation,
	}
}

func (h *createDirectConversationHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request roomin.CreateDirectConversationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.createDirectConversation.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("CreateDirectConversation failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("CreateDirectConversation failed"))
	}
	return result, nil
}
