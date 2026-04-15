// CODE_GENERATOR: registry
package server

import (
	"context"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	paymenthttp "go-socket/core/modules/payment/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

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

func (s *paymentHTTPServer) Stop(_ context.Context) error {
	return nil
}
