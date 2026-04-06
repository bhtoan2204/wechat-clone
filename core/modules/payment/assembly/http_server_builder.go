package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	paymentcommand "go-socket/core/modules/payment/application/command"
	paymentquery "go-socket/core/modules/payment/application/query"
	paymentservice "go-socket/core/modules/payment/application/service"
	paymentrepo "go-socket/core/modules/payment/infra/persistent/repository"
	"go-socket/core/modules/payment/providers"
	mockprovider "go-socket/core/modules/payment/providers/mock"
	stripeprovider "go-socket/core/modules/payment/providers/stripe"
	paymentserver "go-socket/core/modules/payment/transport/server"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	paymentRepos := paymentrepo.NewRepoImpl(appContext)
	intentStore := paymentrepo.NewProviderPaymentRepoImpl(appContext.GetDB())
	providerRegistry := providers.NewProviderRegistry()
	providerRegistry.Register(mockprovider.NewProvider(appContext.GetConfig().LedgerConfig.MockWebhookSecret))
	if stripe := stripeprovider.NewProvider(appContext.GetConfig().LedgerConfig.Stripe); stripe.Enabled() {
		providerRegistry.Register(stripe)
	}
	providerPaymentService := paymentservice.NewPaymentService(intentStore, providerRegistry)

	createPayment := cqrs.NewDispatcher(paymentcommand.NewCreateProviderPaymentHandler(providerPaymentService))
	processWebhook := cqrs.NewDispatcher(paymentcommand.NewProcessProviderWebhookHandler(providerPaymentService))
	deposit := cqrs.NewDispatcher(paymentcommand.NewDepositHandler(paymentRepos))
	rebuildProjection := cqrs.NewDispatcher(paymentcommand.NewRebuildProjectionHandler(paymentRepos))
	transfer := cqrs.NewDispatcher(paymentcommand.NewTransferHandler(paymentRepos))
	withdrawal := cqrs.NewDispatcher(paymentcommand.NewWithdrawalHandler(paymentRepos))
	listTransaction := cqrs.NewDispatcher(paymentquery.NewListTransactionHandler(paymentRepos))

	server, err := paymentserver.NewHTTPServer(
		deposit,
		rebuildProjection,
		transfer,
		withdrawal,
		listTransaction,
		createPayment,
		processWebhook,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
