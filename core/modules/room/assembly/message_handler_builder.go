package assembly

import (
	appCtx "go-socket/core/context"
	"go-socket/core/modules/room/application/messaging"
	roomrepo "go-socket/core/modules/room/infra/persistent/repository"
	"go-socket/core/shared/config"
)

func BuildMessageHandler(cfg *config.Config, appCtx *appCtx.AppContext) (messaging.MessageHandler, error) {
	repos := roomrepo.NewRepoImpl(appCtx)
	return messaging.NewMessageHandler(cfg, repos.RoomAccountProjectionRepository())
}
