// CODE_GENERATOR: handler
package handler

import (
	"errors"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type logoutHandler struct {
	logout cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse]
}

func NewLogoutHandler(logout cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse]) *logoutHandler {
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
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.logout.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("Logout failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("Logout failed"))
	}
	return result, nil
}
