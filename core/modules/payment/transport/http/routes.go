// CODE_GENERATOR - do not edit: routing
package http

import (
	"wechat-clone/core/modules/payment/application/dto/in"
	"wechat-clone/core/modules/payment/application/dto/out"
	"wechat-clone/core/modules/payment/transport/http/handler"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(
	routes *gin.RouterGroup,
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse],
) {
	routes.POST("/payment/webhooks/:provider", httpx.Wrap(handler.NewProcessWebhookHandler(processWebhook)))
}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	createPayment cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse],
) {
	routes.POST("/payment/intents", httpx.Wrap(handler.NewCreatePaymentHandler(createPayment)))
}
