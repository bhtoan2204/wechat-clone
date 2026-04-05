package assembly

import (
	"context"

	appCtx "go-socket/core/context"
	ledgercommand "go-socket/core/modules/ledger/application/command"
	ledgerquery "go-socket/core/modules/ledger/application/query"
	ledgerserver "go-socket/core/modules/ledger/transport/server"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (infrahttp.HTTPServer, error) {
	ledgerService := BuildService(appContext)
	createTransaction := cqrs.NewDispatcher(ledgercommand.NewCreateTransactionHandler(ledgerService))
	getAccountBalance := cqrs.NewDispatcher(ledgerquery.NewGetAccountBalanceHandler(ledgerService))
	getTransaction := cqrs.NewDispatcher(ledgerquery.NewGetTransactionHandler(ledgerService))

	return ledgerserver.NewHTTPServer(createTransaction, getAccountBalance, getTransaction)
}
