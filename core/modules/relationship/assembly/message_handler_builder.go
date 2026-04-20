package assembly

import (
	appCtx "wechat-clone/core/context"
	relationshipmessaging "wechat-clone/core/modules/relationship/application/messaging"
	relationshiprepo "wechat-clone/core/modules/relationship/infra/persistent/repository"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
)

func buildMessagingHandler(cfg *config.Config, appCtx *appCtx.AppContext) (relationshipmessaging.MessageHandler, error) {
	repos := relationshiprepo.NewRepoImpl(appCtx)
	handler, err := relationshipmessaging.NewMessageHandler(cfg, repos)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return handler, nil
}
