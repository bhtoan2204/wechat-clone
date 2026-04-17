package assembly

import (
	appCtx "go-socket/core/context"
	roomprojection "go-socket/core/modules/room/application/messaging"
	roomservice "go-socket/core/modules/room/application/service"
	roomrepo "go-socket/core/modules/room/infra/persistent/repository"
	roomCassandra "go-socket/core/modules/room/infra/projection/cassandra"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"
)

func buildProjectionHandler(cfg *config.Config, appCtx *appCtx.AppContext) (roomprojection.MessageHandler, error) {
	repos, err := roomrepo.NewRepoImpl(appCtx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	roomReadRepos, err := roomCassandra.NewQueryRepoImpl(
		appCtx.GetConfig().CassandraConfig,
		appCtx.GetCassandraSession(),
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	roomService := roomservice.NewService(appCtx, roomReadRepos)
	return roomprojection.NewMessageHandler(cfg, repos, roomService)
}
