package handler

import (
	"errors"
	"go-socket/core/modules/payment/application/command"
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type transferHandler struct {
	commandBus command.Bus
}

func NewTransferHandler(commandBus command.Bus) *transferHandler {
	return &transferHandler{commandBus: commandBus}
}

func (h *transferHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.TransferRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.commandBus.Transfer.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("Transfer failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("transfer failed"))
	}
	return result, nil
}
