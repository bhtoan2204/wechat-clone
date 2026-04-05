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

type getChatPresenceHandler struct {
	getChatPresence cqrs.Dispatcher[*roomin.GetChatPresenceRequest, *roomout.ChatPresenceResponse]
}

func NewGetChatPresenceHandler(getChatPresence cqrs.Dispatcher[*roomin.GetChatPresenceRequest, *roomout.ChatPresenceResponse]) *getChatPresenceHandler {
	return &getChatPresenceHandler{
		getChatPresence: getChatPresence,
	}
}

func (h *getChatPresenceHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.GetChatPresenceRequest{AccountID: c.Param("account_id")}
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.getChatPresence.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetChatPresence failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("GetChatPresence failed"))
	}
	return result, nil
}
