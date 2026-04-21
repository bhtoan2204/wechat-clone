package assembly

import (
	appCtx "wechat-clone/core/context"
	relationshipprocessor "wechat-clone/core/modules/relationship/application/projection/processor"
	relationshipprojection "wechat-clone/core/modules/relationship/infra/projection/cassandra"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
)

func buildProjectionProcessor(cfg *config.Config, appCtx *appCtx.AppContext) (relationshipprocessor.Processor, error) {
	projRepo, err := relationshipprojection.NewProjectionRepo(appCtx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	processor, err := relationshipprocessor.NewProcessor(cfg, projRepo)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return processor, nil
}
