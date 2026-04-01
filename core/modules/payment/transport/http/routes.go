package http

import (
	"go-socket/core/modules/payment/application/command"
	"go-socket/core/modules/payment/application/query"
	"go-socket/core/modules/payment/transport/http/handler"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(routes *gin.RouterGroup, providerPaymentHandler *handler.ProviderPaymentHandler) {
	routes.POST("/payment/webhooks/:provider", providerPaymentHandler.HandleWebhook)
}

func RegisterPrivateRoutes(routes *gin.RouterGroup, commandBus command.Bus, queryBus query.Bus, providerPaymentHandler *handler.ProviderPaymentHandler) {
	routes.POST("/payment/intents", providerPaymentHandler.CreatePayment)
	routes.POST("/payment/deposit", httpx.Wrap(handler.NewDepositHandler(commandBus)))
	routes.POST("/payment/projection/rebuild", httpx.Wrap(handler.NewRebuildProjectionHandler(commandBus)))
	routes.POST("/payment/transfer", httpx.Wrap(handler.NewTransferHandler(commandBus)))
	routes.POST("/payment/withdrawal", httpx.Wrap(handler.NewWithdrawalHandler(commandBus)))

	routes.GET("/payment/transaction", httpx.Wrap(handler.NewListTransactionHandler(queryBus)))
}
