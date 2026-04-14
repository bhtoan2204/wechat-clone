// CODE_GENERATOR - do not edit: routing
package http

import (
	"go-socket/core/modules/ledger/application/dto/in"
	"go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/modules/ledger/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(_ *gin.RouterGroup) {}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	createTransaction cqrs.Dispatcher[*in.CreateTransactionRequest, *out.TransactionResponse],
	getAccountBalance cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse],
	getTransaction cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse],
) {
	routes.POST("/ledger/transactions", httpx.Wrap(handler.NewCreateTransactionHandler(createTransaction)))
	routes.GET("/ledger/accounts/:account_id/balance", httpx.Wrap(handler.NewGetAccountBalanceHandler(getAccountBalance)))
	routes.GET("/ledger/transactions/:transaction_id", httpx.Wrap(handler.NewGetTransactionHandler(getTransaction)))
}
