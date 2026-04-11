package assembly

import (
	appCtx "go-socket/core/context"
	roomprojection "go-socket/core/modules/room/application/projection"
	roominfra "go-socket/core/modules/room/infra/projection"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"
	modruntime "go-socket/core/shared/runtime"
)

func buildServingProjectionProcessor(cfg *config.Config, appCtx *appCtx.AppContext) (modruntime.Module, error) {
	timelineProjector, err := roominfra.NewCassandraTimelineProjector(cfg.CassandraConfig, appCtx.GetCassandraSession())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	searchIndexer, err := roominfra.NewElasticsearchMessageIndexer(cfg.ElasticsearchConfig, appCtx.GetElasticsearchClient())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return roomprojection.NewProcessor(cfg, timelineProjector, searchIndexer)
}
