// CODE_GENERATOR - do not edit: routing
package http

import (
	"wechat-clone/core/modules/ledger/application/dto/in"
	"wechat-clone/core/modules/ledger/application/dto/out"
	"wechat-clone/core/modules/ledger/transport/http/handler"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(_ *gin.RouterGroup) {}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	getAccountBalance cqrs.Dispatcher[*in.GetAccountBalanceRequest, *out.AccountBalanceResponse],
	getTransaction cqrs.Dispatcher[*in.GetTransactionRequest, *out.TransactionResponse],
	transferTransaction cqrs.Dispatcher[*in.TransferTransactionRequest, *out.TransactionTransactionResponse],
	listTransaction cqrs.Dispatcher[*in.ListTransactionRequest, *out.ListTransactionResponse],
) {
	routes.GET("/ledger/wallet/balance", httpx.Wrap(handler.NewGetAccountBalanceHandler(getAccountBalance)))
	routes.GET("/ledger/transactions/:transaction_id", httpx.Wrap(handler.NewGetTransactionHandler(getTransaction)))
	routes.POST("/ledger/transfers", httpx.Wrap(handler.NewTransferTransactionHandler(transferTransaction)))
	routes.GET("/ledger/transactions", httpx.Wrap(handler.NewListTransactionHandler(listTransaction)))
}
