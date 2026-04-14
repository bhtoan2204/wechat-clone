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

type changePasswordHandler struct {
	changePassword cqrs.Dispatcher[*in.ChangePasswordRequest, *out.ChangePasswordResponse]
}

func NewChangePasswordHandler(
	changePassword cqrs.Dispatcher[*in.ChangePasswordRequest, *out.ChangePasswordResponse],
) *changePasswordHandler {
	return &changePasswordHandler{
		changePassword: changePassword,
	}
}

func (h *changePasswordHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.ChangePasswordRequest
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

	result, err := h.changePassword.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("ChangePassword failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
