// CODE_GENERATOR: registry
package server

import (
	"context"

	"wechat-clone/core/modules/payment/application/dto/in"
	"wechat-clone/core/modules/payment/application/dto/out"
	paymenthttp "wechat-clone/core/modules/payment/transport/http"
	"wechat-clone/core/shared/pkg/cqrs"
	infrahttp "wechat-clone/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type paymentHTTPServer struct {
	createPayment  cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse]
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse]
}

func NewHTTPServer(
	createPayment cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse],
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse],
) (infrahttp.HTTPServer, error) {
	return &paymentHTTPServer{
		createPayment:  createPayment,
		processWebhook: processWebhook,
	}, nil
}

func (s *paymentHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPublicRoutes(routes, s.processWebhook)
}

func (s *paymentHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPrivateRoutes(routes, s.createPayment)
}

func (s *paymentHTTPServer) RegisterSocketRoutes(routes *gin.RouterGroup) {
}

func (s *paymentHTTPServer) Stop(ctx context.Context) error {
	return nil
}
