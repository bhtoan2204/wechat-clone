package assembly

import (
	appCtx "go-socket/core/context"
	"go-socket/core/modules/room/application/projection/processor"
	roomCassandra "go-socket/core/modules/room/infra/projection/cassandra"
	roomElasticsearch "go-socket/core/modules/room/infra/projection/elasticsearch"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"
	modruntime "go-socket/core/shared/runtime"
)

func buildServingProjectionProcessor(cfg *config.Config, appCtx *appCtx.AppContext) (modruntime.Module, error) {
	servingProjector, err := roomCassandra.NewCassandraTimelineProjector(cfg.CassandraConfig, appCtx.GetCassandraSession())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	searchIndexer, err := roomElasticsearch.NewElasticsearchMessageIndexer(cfg.ElasticsearchConfig, appCtx.GetElasticsearchClient())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return processor.NewProcessor(cfg, servingProjector, searchIndexer)
}
