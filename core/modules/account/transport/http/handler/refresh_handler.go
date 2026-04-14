// CODE_GENERATOR - do not edit: handler
package handler

import (
	"net/http"

	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type refreshHandler struct {
	refresh cqrs.Dispatcher[*in.RefreshRequest, *out.RefreshResponse]
}

func NewRefreshHandler(
	refresh cqrs.Dispatcher[*in.RefreshRequest, *out.RefreshResponse],
) *refreshHandler {
	return &refreshHandler{
		refresh: refresh,
	}
}

func (h *refreshHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.RefreshRequest
	request.UserAgent = c.GetHeader("User-Agent")
	request.IpAddress = c.GetHeader("X-Forwarded-For")
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

	result, err := h.refresh.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("Refresh failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
