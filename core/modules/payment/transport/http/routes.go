// CODE_GENERATOR - do not edit: routing
package http

import (
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(
	routes *gin.RouterGroup,
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse],
) {
	routes.POST("/payment/webhooks/:provider", httpx.Wrap(handler.NewProcessWebhookHandler(processWebhook)))
}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	deposit cqrs.Dispatcher[*in.DepositRequest, *out.DepositResponse],
	rebuildProjection cqrs.Dispatcher[*in.RebuildProjectionRequest, *out.RebuildProjectionResponse],
	transfer cqrs.Dispatcher[*in.TransferRequest, *out.TransferResponse],
	withdrawal cqrs.Dispatcher[*in.WithdrawalRequest, *out.WithdrawalResponse],
	listTransaction cqrs.Dispatcher[*in.ListTransactionRequest, *out.ListTransactionResponse],
	createPayment cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse],
) {
	routes.POST("/payment/deposit", httpx.Wrap(handler.NewDepositHandler(deposit)))
	routes.POST("/payment/projection/rebuild", httpx.Wrap(handler.NewRebuildProjectionHandler(rebuildProjection)))
	routes.POST("/payment/transfer", httpx.Wrap(handler.NewTransferHandler(transfer)))
	routes.POST("/payment/withdrawal", httpx.Wrap(handler.NewWithdrawalHandler(withdrawal)))
	routes.GET("/payment/transaction", httpx.Wrap(handler.NewListTransactionHandler(listTransaction)))
	routes.POST("/payment/intents", httpx.Wrap(handler.NewCreatePaymentHandler(createPayment)))
}
