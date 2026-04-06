// CODE_GENERATOR: registry
package server

import (
	"context"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	paymenthttp "go-socket/core/modules/payment/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type paymentHTTPServer struct {
	deposit           cqrs.Dispatcher[*in.DepositRequest, *out.DepositResponse]
	rebuildProjection cqrs.Dispatcher[*in.RebuildProjectionRequest, *out.RebuildProjectionResponse]
	transfer          cqrs.Dispatcher[*in.TransferRequest, *out.TransferResponse]
	withdrawal        cqrs.Dispatcher[*in.WithdrawalRequest, *out.WithdrawalResponse]
	listTransaction   cqrs.Dispatcher[*in.ListTransactionRequest, *out.ListTransactionResponse]
	createPayment     cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse]
	processWebhook    cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse]
}

func NewHTTPServer(
	deposit cqrs.Dispatcher[*in.DepositRequest, *out.DepositResponse],
	rebuildProjection cqrs.Dispatcher[*in.RebuildProjectionRequest, *out.RebuildProjectionResponse],
	transfer cqrs.Dispatcher[*in.TransferRequest, *out.TransferResponse],
	withdrawal cqrs.Dispatcher[*in.WithdrawalRequest, *out.WithdrawalResponse],
	listTransaction cqrs.Dispatcher[*in.ListTransactionRequest, *out.ListTransactionResponse],
	createPayment cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse],
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse],
) (infrahttp.HTTPServer, error) {
	return &paymentHTTPServer{
		deposit:           deposit,
		rebuildProjection: rebuildProjection,
		transfer:          transfer,
		withdrawal:        withdrawal,
		listTransaction:   listTransaction,
		createPayment:     createPayment,
		processWebhook:    processWebhook,
	}, nil
}

func (s *paymentHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPublicRoutes(routes, s.processWebhook)
}

func (s *paymentHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPrivateRoutes(routes, s.deposit, s.rebuildProjection, s.transfer, s.withdrawal, s.listTransaction, s.createPayment)
}

func (s *paymentHTTPServer) Stop(_ context.Context) error {
	return nil
}
