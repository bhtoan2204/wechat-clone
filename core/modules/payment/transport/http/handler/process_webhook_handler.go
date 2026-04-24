// CODE_GENERATOR - do not edit: handler
package handler

import (
	"io"
	"net/http"

	"wechat-clone/core/modules/payment/application/dto/in"
	"wechat-clone/core/modules/payment/application/dto/out"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type processWebhookHandler struct {
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse]
}

func NewProcessWebhookHandler(
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse],
) *processWebhookHandler {
	return &processWebhookHandler{
		processWebhook: processWebhook,
	}
}

func (h *processWebhookHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.ProcessWebhookRequest
	request.Provider = c.Param("provider")
	request.Signature = c.GetHeader("Stripe-Signature")

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Errorw("Read request body failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unable to read request body"})
		return nil, nil
	}
	request.Payload = string(payload)

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.processWebhook.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("ProcessWebhook failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
