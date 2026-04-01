package handler

import (
	"errors"
	"net/http"

	"go-socket/core/modules/payment/application/command"
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/aggregate"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type withdrawalHandler struct {
	withdrawal cqrs.Dispatcher[*in.WithdrawalRequest, *out.WithdrawalResponse]
}

func NewWithdrawalHandler(withdrawal cqrs.Dispatcher[*in.WithdrawalRequest, *out.WithdrawalResponse]) *withdrawalHandler {
	return &withdrawalHandler{withdrawal: withdrawal}
}

func (h *withdrawalHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.WithdrawalRequest
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
	result, err := h.withdrawal.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("Withdrawal failed", zap.Error(err))
		switch {
		case errors.Is(err, command.ErrPaymentAccountNotFound):
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return nil, nil
		case errors.Is(err, aggregate.ErrInvalidPaymentAmount):
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return nil, nil
		case errors.Is(err, aggregate.ErrInsufficientBalance):
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
			return nil, nil
		case errors.Is(err, paymentrepos.ErrPaymentVersionConflict):
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
			return nil, nil
		default:
			return nil, stackerr.Error(err)
		}
	}
	return result, nil
}
