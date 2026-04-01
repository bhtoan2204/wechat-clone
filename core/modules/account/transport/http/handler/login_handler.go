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

type loginHandler struct {
	login cqrs.Dispatcher[*in.LoginRequest, *out.LoginResponse]
}

func NewLoginHandler(login cqrs.Dispatcher[*in.LoginRequest, *out.LoginResponse]) *loginHandler {
	return &loginHandler{
		login: login,
	}
}

func (h *loginHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.login.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("Login failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("Login failed"))
	}
	return result, nil
}
