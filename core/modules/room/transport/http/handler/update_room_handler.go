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

type updateRoomHandler struct {
	updateRoom cqrs.Dispatcher[*in.UpdateRoomRequest, *out.UpdateRoomResponse]
}

func NewUpdateRoomHandler(updateRoom cqrs.Dispatcher[*in.UpdateRoomRequest, *out.UpdateRoomResponse]) *updateRoomHandler {
	return &updateRoomHandler{
		updateRoom: updateRoom,
	}
}

func (h *updateRoomHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.UpdateRoomRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.updateRoom.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("UpdateRoom failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("UpdateRoom failed"))
	}
	return result, nil
}
