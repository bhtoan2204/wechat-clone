// CODE_GENERATOR - do not edit: handler
package handler

import (
	"net/http"

	"wechat-clone/core/modules/ledger/application/dto/in"
	"wechat-clone/core/modules/ledger/application/dto/out"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

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
	var request in.GetAccountBalanceRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.getAccountBalance.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetAccountBalance failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
