package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	paymentcommand "go-socket/core/modules/payment/application/command"
	paymentservice "go-socket/core/modules/payment/application/service"
	paymentrepo "go-socket/core/modules/payment/infra/persistent/repository"
	provideradapter "go-socket/core/modules/payment/infra/provider"
	"go-socket/core/modules/payment/providers"
	mockprovider "go-socket/core/modules/payment/providers/mock"
	stripeprovider "go-socket/core/modules/payment/providers/stripe"
	paymentserver "go-socket/core/modules/payment/transport/server"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func buildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	paymentRepos := paymentrepo.NewRepoImpl(appContext)
	providerRegistry := providers.NewProviderRegistry()
	providerRegistry.Register(mockprovider.NewProvider(appContext.GetConfig().LedgerConfig.MockWebhookSecret))
	if stripe := stripeprovider.NewProvider(appContext.GetConfig().LedgerConfig.Stripe); stripe.Enabled() {
		providerRegistry.Register(stripe)
	}
	paymentCommandService := paymentservice.NewPaymentCommandService(appContext, paymentRepos, provideradapter.NewPaymentProviderRegistry(providerRegistry))

	createPayment := cqrs.NewDispatcher(paymentcommand.NewCreatePayment(paymentCommandService))
	processWebhook := cqrs.NewDispatcher(paymentcommand.NewProcessWebhook(paymentCommandService))

	server, err := paymentserver.NewHTTPServer(
		createPayment,
		processWebhook,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
