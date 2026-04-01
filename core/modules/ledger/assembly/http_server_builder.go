package assembly

import (
	"context"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/ledger/transport/http/handler"
	ledgerserver "go-socket/core/modules/ledger/transport/server"
	infrahttp "go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (infrahttp.HTTPServer, error) {
	ledgerService := BuildService(appContext)
	ledgerHandler := handler.NewLedgerHandler(ledgerService)

	return ledgerserver.NewHTTPServer(ledgerHandler)
}
