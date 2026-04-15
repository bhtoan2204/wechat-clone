package assembly

import (
	appCtx "go-socket/core/context"
	ledgermessaging "go-socket/core/modules/ledger/application/messaging"
	ledgerrepo "go-socket/core/modules/ledger/infra/persistent/repository"
	ledgerserver "go-socket/core/modules/ledger/transport/server"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"
	modruntime "go-socket/core/shared/runtime"
)

func buildMessagingRuntime(cfg *config.Config, appCtx *appCtx.AppContext) (modruntime.Module, error) {
	messageHandler, err := ledgermessaging.NewMessageHandler(
		cfg,
		BuildService(appCtx),
		ledgerrepo.NewLedgerProjectionRepoImpl(appCtx.GetDB()),
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return ledgerserver.NewServer(messageHandler)
}
