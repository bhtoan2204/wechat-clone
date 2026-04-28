package assembly

import (
	appCtx "wechat-clone/core/context"
	roomprojection "wechat-clone/core/modules/room/application/messaging"
	roomservice "wechat-clone/core/modules/room/application/service"
	roomrepo "wechat-clone/core/modules/room/infra/persistent/repository"
	roomCassandra "wechat-clone/core/modules/room/infra/projection/cassandra"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
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
	accountProjectionRepo := roomrepo.NewRoomAccountImpl(appCtx.GetDB())
	roomService := roomservice.NewService(appCtx, roomReadRepos)
	return roomprojection.NewMessageHandler(cfg, repos, accountProjectionRepo, roomService)
}
