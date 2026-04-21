// CODE_GENERATOR: assembly
package assembly

import (
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/config"
	modruntime "wechat-clone/core/shared/runtime"
)

func BuildProjectionRuntime(cfg *config.Config, appContext *appCtx.AppContext) (modruntime.Module, error) {
	return buildProjectionRuntime(cfg, appContext)
}
