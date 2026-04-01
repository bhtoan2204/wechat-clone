package assembly

import (
	"context"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/ledger/application/service"
	ledgerrepo "go-socket/core/modules/ledger/infra/persistent/repository"
	"go-socket/core/modules/ledger/providers"
	"go-socket/core/modules/ledger/providers/mock"
	stripeprovider "go-socket/core/modules/ledger/providers/stripe"
	"go-socket/core/modules/ledger/transport/http/handler"
	ledgerserver "go-socket/core/modules/ledger/transport/server"
	infrahttp "go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (infrahttp.HTTPServer, error) {
	repos := ledgerrepo.NewRepoImpl(appContext)

	providerRegistry := providers.NewProviderRegistry()
	providerRegistry.Register(mock.NewProvider(appContext.GetConfig().LedgerConfig.MockWebhookSecret))
	if stripe := stripeprovider.NewProvider(appContext.GetConfig().LedgerConfig.Stripe); stripe.Enabled() {
		providerRegistry.Register(stripe)
	}

	ledgerService := service.NewLedgerService(repos)
	paymentService := service.NewPaymentService(repos, ledgerService, providerRegistry)

	ledgerHandler := handler.NewLedgerHandler(ledgerService)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	return ledgerserver.NewHTTPServer(ledgerHandler, paymentHandler)
}
