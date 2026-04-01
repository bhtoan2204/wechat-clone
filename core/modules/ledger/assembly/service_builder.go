package assembly

import (
	appCtx "go-socket/core/context"
	"go-socket/core/modules/ledger/application/service"
	ledgerrepo "go-socket/core/modules/ledger/infra/persistent/repository"
)

func BuildService(appContext *appCtx.AppContext) *service.LedgerService {
	return service.NewLedgerService(ledgerrepo.NewRepoImpl(appContext))
}
