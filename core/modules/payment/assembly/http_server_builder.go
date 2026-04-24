package assembly

import (
	"context"

	appCtx "wechat-clone/core/context"
	paymentcommand "wechat-clone/core/modules/payment/application/command"
	paymentservice "wechat-clone/core/modules/payment/application/service"
	paymentrepo "wechat-clone/core/modules/payment/infra/persistent/repository"
	provideradapter "wechat-clone/core/modules/payment/infra/provider"
	"wechat-clone/core/modules/payment/providers"
	mockprovider "wechat-clone/core/modules/payment/providers/mock"
	stripeprovider "wechat-clone/core/modules/payment/providers/stripe"
	paymentserver "wechat-clone/core/modules/payment/transport/server"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
	infrahttp "wechat-clone/core/shared/transport/http"
)

func buildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (infrahttp.HTTPServer, error) {
	paymentRepos := paymentrepo.NewRepoImpl(appContext)
	providerRegistry := providers.NewProviderRegistry()
	providerRegistry.Register(mockprovider.NewProvider(appContext.GetConfig().LedgerConfig.MockWebhookSecret))
	if stripe := stripeprovider.NewProvider(appContext.GetConfig().LedgerConfig.Stripe); stripe.Enabled() {
		providerRegistry.Register(stripe)
	}

	paymentCommandService := paymentservice.NewPaymentCommandService(appContext, paymentRepos, provideradapter.NewPaymentProviderRegistry(providerRegistry))
	createPayment := cqrs.NewDispatcher(paymentcommand.NewCreatePayment(paymentCommandService))
	processWebhook := cqrs.NewDispatcher(paymentcommand.NewProcessWebhook(paymentCommandService))

	server, err := paymentserver.NewHTTPServer(createPayment, processWebhook)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
