// CODE_GENERATOR - do not edit: handler
package handler

import (
	"net/http"

	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type createTransactionHandler struct {
	createTransaction cqrs.Dispatcher[*in.CreateTransactionRequest, *out.TransactionResponse]
}

func NewCreateTransactionHandler(
	createTransaction cqrs.Dispatcher[*in.CreateTransactionRequest, *out.TransactionResponse],
) *createTransactionHandler {
	return &createTransactionHandler{
		createTransaction: createTransaction,
	}
}

func (h *createTransactionHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.CreateTransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.createTransaction.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("CreateTransaction failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
