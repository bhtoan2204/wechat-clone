package handler

import (
	"errors"
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type listTransactionHandler struct {
	listTransaction cqrs.Dispatcher[*in.ListTransactionRequest, *out.ListTransactionResponse]
}

func NewListTransactionHandler(listTransaction cqrs.Dispatcher[*in.ListTransactionRequest, *out.ListTransactionResponse]) *listTransactionHandler {
	return &listTransactionHandler{listTransaction: listTransaction}
}

func (h *listTransactionHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)

	var request in.ListTransactionRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}
	result, err := h.listTransaction.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("List failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("list failed"))
	}
	return result, nil
}
