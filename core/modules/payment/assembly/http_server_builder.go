package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	paymentserver "go-socket/core/modules/payment/transport/server"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	commandBus := BuildBuses(appContext)
	return paymentserver.NewHTTPServer(commandBus)
}
