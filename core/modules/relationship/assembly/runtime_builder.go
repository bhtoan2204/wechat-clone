package assembly

import (
	appCtx "wechat-clone/core/context"
	relationshipserver "wechat-clone/core/modules/relationship/transport/server"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
	modruntime "wechat-clone/core/shared/runtime"
)

func buildMessagingRuntime(cfg *config.Config, appCtx *appCtx.AppContext) (modruntime.Module, error) {
	messageHandler, err := buildMessagingHandler(cfg, appCtx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return relationshipserver.NewServer(messageHandler)
}
