package http

import (
	"go-socket/core/modules/ledger/transport/http/handler"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(_ *gin.RouterGroup) {}

func RegisterPrivateRoutes(routes *gin.RouterGroup, ledgerHandler *handler.LedgerHandler) {
	routes.POST("/ledger/transactions", ledgerHandler.CreateTransaction)
	routes.GET("/ledger/accounts/:account_id/balance", ledgerHandler.GetAccountBalance)
	routes.GET("/ledger/transactions/:transaction_id", ledgerHandler.GetTransaction)
}
