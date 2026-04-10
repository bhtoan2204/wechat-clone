package assembly

import (
	appCtx "go-socket/core/context"
	ledgermessaging "go-socket/core/modules/ledger/application/messaging"
	ledgerserver "go-socket/core/modules/ledger/transport/server"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"
)

func BuildServer(cfg *config.Config, appCtx *appCtx.AppContext) (ledgerserver.Server, error) {
	messageHandler, err := ledgermessaging.NewMessageHandler(cfg, BuildService(appCtx))
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return ledgerserver.NewServer(messageHandler)
}
