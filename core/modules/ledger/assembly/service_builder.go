package assembly

import (
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/ledger/application/service"
	ledgerrepo "wechat-clone/core/modules/ledger/infra/persistent/repository"
	ledgerprojection "wechat-clone/core/modules/ledger/infra/projection"
)

func BuildService(appContext *appCtx.AppContext) service.LedgerService {
	return service.NewLedgerService(ledgerrepo.NewRepoImpl(appContext))
}

func BuildQueryService(appContext *appCtx.AppContext) service.LedgerQueryService {
	readRepo, err := ledgerprojection.NewLedgerReadRepository(appContext.GetDB())
	if err != nil {
		panic(err)
	}
	return service.NewLedgerQueryService(readRepo)
}
