// CODE_GENERATOR: assembly
package assembly

import (
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
	modruntime "wechat-clone/core/shared/runtime"
)

func BuildMessagingRuntime(cfg *config.Config, appContext *appCtx.AppContext) (modruntime.Module, error) {
	runtime, err := buildMessagingRuntime(cfg, appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return runtime, nil
}
