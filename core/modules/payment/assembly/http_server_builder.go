package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	paymentcommand "go-socket/core/modules/payment/application/command"
	paymentquery "go-socket/core/modules/payment/application/query"
	paymentservice "go-socket/core/modules/payment/application/service"
	paymentrepo "go-socket/core/modules/payment/infra/persistent/repository"
	"go-socket/core/modules/payment/providers"
	"go-socket/core/modules/payment/providers/mock"
	stripeprovider "go-socket/core/modules/payment/providers/stripe"
	"go-socket/core/modules/payment/transport/http/handler"
	paymentserver "go-socket/core/modules/payment/transport/server"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	paymentRepos := paymentrepo.NewRepoImpl(appContext)

	deposit := cqrs.NewDispatcher(paymentcommand.NewDepositHandler(paymentRepos))
	rebuildProjection := cqrs.NewDispatcher(paymentcommand.NewRebuildProjectionHandler(paymentRepos))
	transfer := cqrs.NewDispatcher(paymentcommand.NewTransferHandler(paymentRepos))
	withdrawal := cqrs.NewDispatcher(paymentcommand.NewWithdrawalHandler(paymentRepos))
	listTransaction := cqrs.NewDispatcher(paymentquery.NewListTransactionHandler(paymentRepos))

	intentStore := paymentrepo.NewProviderPaymentRepoImpl(appContext.GetDB())
	registeredGateway, ok := appContext.GetService("payment.ledger_gateway")
	if !ok {
		return nil, ErrMissingLedgerGateway
	}
	ledgerGateway, ok := registeredGateway.(paymentservice.LedgerGateway)
	if !ok {
		return nil, ErrInvalidLedgerGateway
	}

	providerRegistry := providers.NewProviderRegistry()
	providerRegistry.Register(mock.NewProvider(appContext.GetConfig().LedgerConfig.MockWebhookSecret))
	if stripe := stripeprovider.NewProvider(appContext.GetConfig().LedgerConfig.Stripe); stripe.Enabled() {
		providerRegistry.Register(stripe)
	}

	providerPaymentService := paymentservice.NewPaymentService(intentStore, ledgerGateway, providerRegistry)
	providerPaymentHandler := handler.NewProviderPaymentHandler(providerPaymentService)

	return paymentserver.NewHTTPServer(deposit, rebuildProjection, transfer, withdrawal, listTransaction, providerPaymentHandler)
}
