package assembly

import (
	appCtx "go-socket/core/context"
	notificationmessaging "go-socket/core/modules/notification/application/messaging"
	notificationrepo "go-socket/core/modules/notification/infra/persistent/repository"
	"go-socket/core/shared/config"
)

func BuildMessageHandler(cfg *config.Config, appCtx *appCtx.AppContext) (notificationmessaging.MessageHandler, error) {
	repos := notificationrepo.NewRepoImpl(appCtx)
	return notificationmessaging.NewMessageHandler(cfg, appCtx.GetSMTP(), repos.NotificationRepository())
}
