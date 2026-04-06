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

type createRoomHandler struct {
	createRoom cqrs.Dispatcher[*in.CreateRoomRequest, *out.CreateRoomResponse]
}

func NewCreateRoomHandler(
	createRoom cqrs.Dispatcher[*in.CreateRoomRequest, *out.CreateRoomResponse],
) *createRoomHandler {
	return &createRoomHandler{
		createRoom: createRoom,
	}
}

func (h *createRoomHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.CreateRoomRequest
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

	result, err := h.createRoom.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("CreateRoom failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
