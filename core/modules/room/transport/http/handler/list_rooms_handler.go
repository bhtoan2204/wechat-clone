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

type listRoomsHandler struct {
	listRoom cqrs.Dispatcher[*in.ListRoomsRequest, *out.ListRoomsResponse]
}

func NewListRoomsHandler(listRoom cqrs.Dispatcher[*in.ListRoomsRequest, *out.ListRoomsResponse]) *listRoomsHandler {
	return &listRoomsHandler{
		listRoom: listRoom,
	}
}

func (h *listRoomsHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.ListRoomsRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.listRoom.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("ListRooms failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("ListRooms failed"))
	}
	return result, nil
}
