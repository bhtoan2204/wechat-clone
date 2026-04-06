// CODE_GENERATOR: handler
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

type logoutHandler struct {
	logout cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse]
}

func NewLogoutHandler(
	logout cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse],
) *logoutHandler {
	return &logoutHandler{
		logout: logout,
	}
}

func (h *logoutHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.LogoutRequest
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

	result, err := h.logout.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("Logout failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
