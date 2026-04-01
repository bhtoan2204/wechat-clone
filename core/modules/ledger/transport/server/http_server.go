package server

import (
	"context"

	"go-socket/core/modules/ledger/transport/http"
	"go-socket/core/modules/ledger/transport/http/handler"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type ledgerHTTPServer struct {
	ledgerHandler  *handler.LedgerHandler
	paymentHandler *handler.PaymentHandler
}

func NewHTTPServer(ledgerHandler *handler.LedgerHandler, paymentHandler *handler.PaymentHandler) (infrahttp.HTTPServer, error) {
	return &ledgerHTTPServer{
		ledgerHandler:  ledgerHandler,
		paymentHandler: paymentHandler,
	}, nil
}

func (s *ledgerHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	http.RegisterPublicRoutes(routes, s.paymentHandler)
}

func (s *ledgerHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	http.RegisterPrivateRoutes(routes, s.ledgerHandler, s.paymentHandler)
}

func (s *ledgerHTTPServer) Stop(_ context.Context) error {
	return nil
}
