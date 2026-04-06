// CODE_GENERATOR: handler
package handler

import (
	"net/http"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type deleteRoomHandler struct {
	deleteRoom cqrs.Dispatcher[*in.DeleteRoomRequest, *out.DeleteRoomResponse]
}

func NewDeleteRoomHandler(
	deleteRoom cqrs.Dispatcher[*in.DeleteRoomRequest, *out.DeleteRoomResponse],
) *deleteRoomHandler {
	return &deleteRoomHandler{
		deleteRoom: deleteRoom,
	}
}

func (h *deleteRoomHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.DeleteRoomRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	result, err := h.deleteRoom.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("DeleteRoom failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
