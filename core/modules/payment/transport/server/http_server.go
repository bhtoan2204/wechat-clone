package server

import (
	"context"
	"go-socket/core/modules/payment/application/command"
	paymenthttp "go-socket/core/modules/payment/transport/http"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type paymentHTTPServer struct {
	commandBus command.Bus
}

func NewHTTPServer(commandBus command.Bus) (infrahttp.HTTPServer, error) {
	return &paymentHTTPServer{commandBus: commandBus}, nil
}

func (s *paymentHTTPServer) RegisterPublicRoutes(_ *gin.RouterGroup) {}

func (s *paymentHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPrivateRoutes(routes, s.commandBus)
}

func (s *paymentHTTPServer) Stop(_ context.Context) error {
	return nil
}
