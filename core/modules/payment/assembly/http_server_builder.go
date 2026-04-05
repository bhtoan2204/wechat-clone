package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	paymentcommand "go-socket/core/modules/payment/application/command"
	paymentquery "go-socket/core/modules/payment/application/query"
	paymentrepo "go-socket/core/modules/payment/infra/persistent/repository"
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

	// intentStore := paymentrepo.NewProviderPaymentRepoImpl(appContext.GetDB())

	// providerRegistry := providers.NewProviderRegistry()
	// providerRegistry.Register(mock.NewProvider(appContext.GetConfig().LedgerConfig.MockWebhookSecret))
	// if stripe := stripeprovider.NewProvider(appContext.GetConfig().LedgerConfig.Stripe); stripe.Enabled() {
	// 	providerRegistry.Register(stripe)
	// }

	// providerPaymentService := paymentservice.NewPaymentService(intentStore, providerRegistry)
	// createProviderPayment := cqrs.NewDispatcher(paymentcommand.NewCreateProviderPaymentHandler(providerPaymentService))
	// processProviderWebhook := cqrs.NewDispatcher(paymentcommand.NewProcessProviderWebhookHandler(providerPaymentService))
	// providerPaymentHandler := handler.NewProviderPaymentHandler(createProviderPayment, processProviderWebhook)

	return paymentserver.NewHTTPServer(deposit, rebuildProjection, transfer, withdrawal, listTransaction, nil)
}
