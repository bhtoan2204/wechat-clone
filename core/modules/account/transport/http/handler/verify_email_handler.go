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

type verifyEmailHandler struct {
	verifyEmail cqrs.Dispatcher[*in.VerifyEmailRequest, *out.VerifyEmailResponse]
}

func NewVerifyEmailHandler(
	verifyEmail cqrs.Dispatcher[*in.VerifyEmailRequest, *out.VerifyEmailResponse],
) *verifyEmailHandler {
	return &verifyEmailHandler{
		verifyEmail: verifyEmail,
	}
}

func (h *verifyEmailHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.VerifyEmailRequest

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.verifyEmail.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("VerifyEmail failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
