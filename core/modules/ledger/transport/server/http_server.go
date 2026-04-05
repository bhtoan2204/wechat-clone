package server

import (
	"context"

	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/modules/ledger/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type ledgerHTTPServer struct {
	createTransactionHandler cqrs.Dispatcher[*in.CreateTransactionRequest, *out.TransactionResponse]
	getAccountBalanceHandler cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse]
	getTransactionHandler    cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse]
}

func NewHTTPServer(
	createTransactionHandler cqrs.Dispatcher[*in.CreateTransactionRequest, *out.TransactionResponse],
	getAccountBalanceHandler cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse],
	getTransactionHandler cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse]) (infrahttp.HTTPServer, error) {
	return &ledgerHTTPServer{
		createTransactionHandler: createTransactionHandler,
		getAccountBalanceHandler: getAccountBalanceHandler,
		getTransactionHandler:    getTransactionHandler,
	}, nil
}

func (s *ledgerHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	http.RegisterPublicRoutes(routes)
}

func (s *ledgerHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	http.RegisterPrivateRoutes(routes, s.createTransactionHandler, s.getAccountBalanceHandler, s.getTransactionHandler)
}

func (s *ledgerHTTPServer) Stop(_ context.Context) error {
	return nil
}
