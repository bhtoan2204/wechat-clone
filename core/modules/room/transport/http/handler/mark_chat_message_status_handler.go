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

type markChatMessageStatusHandler struct {
	markChatMessageStatus cqrs.Dispatcher[*roomin.MarkChatMessageStatusRequest, *roomout.MarkChatMessageStatusResponse]
}

func NewMarkChatMessageStatusHandler(markChatMessageStatus cqrs.Dispatcher[*roomin.MarkChatMessageStatusRequest, *roomout.MarkChatMessageStatusResponse]) *markChatMessageStatusHandler {
	return &markChatMessageStatusHandler{
		markChatMessageStatus: markChatMessageStatus,
	}
}

func (h *markChatMessageStatusHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.MarkChatMessageStatusRequest{MessageID: c.Param("message_id")}
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.markChatMessageStatus.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("MarkChatMessageStatus failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("MarkChatMessageStatus failed"))
	}
	return result, nil
}
