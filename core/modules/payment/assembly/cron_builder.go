// CODE_GENERATOR: assembly
package assembly

import (
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/config"
	modruntime "wechat-clone/core/shared/runtime"
)

func BuildCronRuntime(cfg *config.Config, appContext *appCtx.AppContext) (modruntime.Module, error) {
	return buildCronRuntime(cfg, appContext)
}
