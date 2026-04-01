package server

import (
	"context"

	"go-socket/core/modules/ledger/transport/http"
	"go-socket/core/modules/ledger/transport/http/handler"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type ledgerHTTPServer struct {
	ledgerHandler *handler.LedgerHandler
}

func NewHTTPServer(ledgerHandler *handler.LedgerHandler) (infrahttp.HTTPServer, error) {
	return &ledgerHTTPServer{
		ledgerHandler: ledgerHandler,
	}, nil
}

func (s *ledgerHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	http.RegisterPublicRoutes(routes)
}

func (s *ledgerHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	http.RegisterPrivateRoutes(routes, s.ledgerHandler)
}

func (s *ledgerHTTPServer) Stop(_ context.Context) error {
	return nil
}
