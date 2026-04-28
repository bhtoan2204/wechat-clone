package assembly

import (
	appCtx "wechat-clone/core/context"
	relationshipmessaging "wechat-clone/core/modules/relationship/application/messaging"
	relationshiprepo "wechat-clone/core/modules/relationship/infra/persistent/repository"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
)

func buildMessagingHandler(cfg *config.Config, appCtx *appCtx.AppContext) (relationshipmessaging.MessageHandler, error) {
	accountRepo := relationshiprepo.NewRelationshipAccountRepo(appCtx.GetDB())
	handler, err := relationshipmessaging.NewMessageHandler(cfg, accountRepo)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return handler, nil
}
