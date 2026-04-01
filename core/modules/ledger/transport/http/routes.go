package http

import (
	"go-socket/core/modules/ledger/transport/http/handler"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(routes *gin.RouterGroup, paymentHandler *handler.PaymentHandler) {
	routes.POST("/ledger/payments/webhooks/:provider", paymentHandler.HandleWebhook)
}

func RegisterPrivateRoutes(routes *gin.RouterGroup, ledgerHandler *handler.LedgerHandler, paymentHandler *handler.PaymentHandler) {
	routes.POST("/ledger/transactions", ledgerHandler.CreateTransaction)
	routes.GET("/ledger/accounts/:account_id/balance", ledgerHandler.GetAccountBalance)
	routes.GET("/ledger/transactions/:transaction_id", ledgerHandler.GetTransaction)
	routes.POST("/ledger/payments", paymentHandler.CreatePayment)
}
