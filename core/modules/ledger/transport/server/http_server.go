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
	getAccountBalance   cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse]
	getTransaction      cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse]
	transferTransaction cqrs.Dispatcher[*in.TransferTransactionRequest, *out.TransactionTransactionResponse]
}

func NewHTTPServer(
	getAccountBalance cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse],
	getTransaction cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse],
	transferTransaction cqrs.Dispatcher[*in.TransferTransactionRequest, *out.TransactionTransactionResponse],
) (infrahttp.HTTPServer, error) {
	return &ledgerHTTPServer{
		getAccountBalance:   getAccountBalance,
		getTransaction:      getTransaction,
		transferTransaction: transferTransaction,
	}, nil
}

func (s *ledgerHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	ledgerhttp.RegisterPublicRoutes(routes)
}

func (s *ledgerHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	ledgerhttp.RegisterPrivateRoutes(routes, s.getAccountBalance, s.getTransaction, s.transferTransaction)
}

func (s *ledgerHTTPServer) Stop(_ context.Context) error {
	return nil
}
