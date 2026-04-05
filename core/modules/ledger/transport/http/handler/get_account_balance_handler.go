// CODE_GENERATOR: handler
package handler

import (
	"net/http"

	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type getAccountBalanceHandler struct {
	getAccountBalance cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse]
}

func NewGetAccountBalanceHandler(
	getAccountBalance cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse],
) *getAccountBalanceHandler {
	return &getAccountBalanceHandler{
		getAccountBalance: getAccountBalance,
	}
}

func (h *getAccountBalanceHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := in.GetAccountBalanceRequest{AccountId: c.Param("account_id")}
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

	result, err := h.getAccountBalance.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetAccountBalance failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return result, nil
}
