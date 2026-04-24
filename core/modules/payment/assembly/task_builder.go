// CODE_GENERATOR: assembly
package assembly

import (
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/config"
	modruntime "wechat-clone/core/shared/runtime"
)

func BuildTaskRuntime(cfg *config.Config, appContext *appCtx.AppContext) (modruntime.Module, error) {
	return buildTaskRuntime(cfg, appContext)
}
