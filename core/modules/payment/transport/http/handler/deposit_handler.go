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

type depositHandler struct {
	commandBus command.Bus
}

func NewDepositHandler(commandBus command.Bus) *depositHandler {
	return &depositHandler{commandBus: commandBus}
}

func (h *depositHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)

	var request in.DepositRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}

	result, err := h.commandBus.Deposit.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("Deposit failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("deposit failed"))
	}

	return result, nil
}
