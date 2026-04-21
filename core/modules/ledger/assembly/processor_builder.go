package assembly

import (
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/ledger/application/projection/processor"
	"wechat-clone/core/modules/ledger/infra/projection"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
	modruntime "wechat-clone/core/shared/runtime"
)

func buildServingProjectionProcessor(cfg *config.Config, appCtx *appCtx.AppContext) (modruntime.Module, error) {
	projector, err := projection.NewLedgerProjector(appCtx.GetDB())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return processor.NewProcessor(cfg, projector)
}
