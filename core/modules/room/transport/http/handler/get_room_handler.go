// CODE_GENERATOR: handler
package handler

import (
	"errors"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type getRoomHandler struct {
	getRoom cqrs.Dispatcher[*in.GetRoomRequest, *out.GetRoomResponse]
}

func NewGetRoomHandler(getRoom cqrs.Dispatcher[*in.GetRoomRequest, *out.GetRoomResponse]) *getRoomHandler {
	return &getRoomHandler{
		getRoom: getRoom,
	}
}

func (h *getRoomHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.GetRoomRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.getRoom.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetRoom failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("GetRoom failed"))
	}
	return result, nil
}
