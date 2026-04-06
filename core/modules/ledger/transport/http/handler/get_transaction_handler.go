// CODE_GENERATOR: handler
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

type getTransactionHandler struct {
	getTransaction cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse]
}

func NewGetTransactionHandler(
	getTransaction cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse],
) *getTransactionHandler {
	return &getTransactionHandler{
		getTransaction: getTransaction,
	}
}

func (h *getTransactionHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.GetTransactionRequest
	request.TransactionID = c.Param("transaction_id")

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	result, err := h.getTransaction.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetTransaction failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
