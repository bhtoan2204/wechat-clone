// CODE_GENERATOR: registry
package server

import (
	"context"

	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/dto/out"
	ledgerhttp "go-socket/core/modules/ledger/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type ledgerHTTPServer struct {
	createTransaction cqrs.Dispatcher[*in.CreateTransactionRequest, *out.TransactionResponse]
	getAccountBalance cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse]
	getTransaction    cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse]
}

func NewHTTPServer(
	createTransaction cqrs.Dispatcher[*in.CreateTransactionRequest, *out.TransactionResponse],
	getAccountBalance cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse],
	getTransaction cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse],
) (infrahttp.HTTPServer, error) {
	return &ledgerHTTPServer{
		createTransaction: createTransaction,
		getAccountBalance: getAccountBalance,
		getTransaction:    getTransaction,
	}, nil
}

func (s *ledgerHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	ledgerhttp.RegisterPublicRoutes(routes)
}

func (s *ledgerHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	ledgerhttp.RegisterPrivateRoutes(routes, s.createTransaction, s.getAccountBalance, s.getTransaction)
}

func (s *ledgerHTTPServer) Stop(_ context.Context) error {
	return nil
}
