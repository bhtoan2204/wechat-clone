package server

import (
	"context"
	paymentin "go-socket/core/modules/payment/application/dto/in"
	paymentout "go-socket/core/modules/payment/application/dto/out"
	paymenthttp "go-socket/core/modules/payment/transport/http"
	"go-socket/core/modules/payment/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type paymentHTTPServer struct {
	deposit                cqrs.Dispatcher[*paymentin.DepositRequest, *paymentout.DepositResponse]
	rebuildProjection      cqrs.Dispatcher[*paymentin.RebuildProjectionRequest, *paymentout.RebuildProjectionResponse]
	transfer               cqrs.Dispatcher[*paymentin.TransferRequest, *paymentout.TransferResponse]
	withdrawal             cqrs.Dispatcher[*paymentin.WithdrawalRequest, *paymentout.WithdrawalResponse]
	listTransaction        cqrs.Dispatcher[*paymentin.ListTransactionRequest, *paymentout.ListTransactionResponse]
	providerPaymentHandler *handler.ProviderPaymentHandler
}

func NewHTTPServer(
	deposit cqrs.Dispatcher[*paymentin.DepositRequest, *paymentout.DepositResponse],
	rebuildProjection cqrs.Dispatcher[*paymentin.RebuildProjectionRequest, *paymentout.RebuildProjectionResponse],
	transfer cqrs.Dispatcher[*paymentin.TransferRequest, *paymentout.TransferResponse],
	withdrawal cqrs.Dispatcher[*paymentin.WithdrawalRequest, *paymentout.WithdrawalResponse],
	listTransaction cqrs.Dispatcher[*paymentin.ListTransactionRequest, *paymentout.ListTransactionResponse],
	providerPaymentHandler *handler.ProviderPaymentHandler,
) (infrahttp.HTTPServer, error) {
	return &paymentHTTPServer{
		deposit:                deposit,
		rebuildProjection:      rebuildProjection,
		transfer:               transfer,
		withdrawal:             withdrawal,
		listTransaction:        listTransaction,
		providerPaymentHandler: providerPaymentHandler,
	}, nil
}

func (s *paymentHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPublicRoutes(routes, s.providerPaymentHandler)
}

func (s *paymentHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	paymenthttp.RegisterPrivateRoutes(routes, s.deposit, s.rebuildProjection, s.transfer, s.withdrawal, s.listTransaction, s.providerPaymentHandler)
}

func (s *paymentHTTPServer) Stop(_ context.Context) error {
	return nil
}
