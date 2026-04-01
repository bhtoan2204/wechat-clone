package http

import (
	paymentin "go-socket/core/modules/payment/application/dto/in"
	paymentout "go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(routes *gin.RouterGroup, providerPaymentHandler *handler.ProviderPaymentHandler) {
	routes.POST("/payment/webhooks/:provider", providerPaymentHandler.HandleWebhook)
}

func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	deposit cqrs.Dispatcher[*paymentin.DepositRequest, *paymentout.DepositResponse],
	rebuildProjection cqrs.Dispatcher[*paymentin.RebuildProjectionRequest, *paymentout.RebuildProjectionResponse],
	transfer cqrs.Dispatcher[*paymentin.TransferRequest, *paymentout.TransferResponse],
	withdrawal cqrs.Dispatcher[*paymentin.WithdrawalRequest, *paymentout.WithdrawalResponse],
	listTransaction cqrs.Dispatcher[*paymentin.ListTransactionRequest, *paymentout.ListTransactionResponse],
	providerPaymentHandler *handler.ProviderPaymentHandler,
) {
	routes.POST("/payment/intents", providerPaymentHandler.CreatePayment)
	routes.POST("/payment/deposit", httpx.Wrap(handler.NewDepositHandler(deposit)))
	routes.POST("/payment/projection/rebuild", httpx.Wrap(handler.NewRebuildProjectionHandler(rebuildProjection)))
	routes.POST("/payment/transfer", httpx.Wrap(handler.NewTransferHandler(transfer)))
	routes.POST("/payment/withdrawal", httpx.Wrap(handler.NewWithdrawalHandler(withdrawal)))

	routes.GET("/payment/transaction", httpx.Wrap(handler.NewListTransactionHandler(listTransaction)))
}
