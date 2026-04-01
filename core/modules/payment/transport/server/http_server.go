package server

import (
	"context"
	"go-socket/core/modules/payment/application/command"
	"go-socket/core/modules/payment/application/query"
	paymenthttp "go-socket/core/modules/payment/transport/http"
	"go-socket/core/modules/payment/transport/http/handler"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type paymentHTTPServer struct {
	commandBus             command.Bus
	queryBus               query.Bus
	providerPaymentHandler *handler.ProviderPaymentHandler
}

func NewHTTPServer(commandBus command.Bus, queryBus query.Bus, providerPaymentHandler *handler.ProviderPaymentHandler) (infrahttp.HTTPServer, error) {
	return &paymentHTTPServer{
		commandBus:             commandBus,
		queryBus:               queryBus,
		providerPaymentHandler: providerPaymentHandler,
	}, nil
}

func (s *paymentHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPublicRoutes(routes, s.providerPaymentHandler)
}

func (s *paymentHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPrivateRoutes(routes, s.commandBus, s.queryBus, s.providerPaymentHandler)
}

func (s *paymentHTTPServer) Stop(_ context.Context) error {
	return nil
}
