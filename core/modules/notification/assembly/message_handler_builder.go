package assembly

import (
	appCtx "go-socket/core/context"
	notificationmessaging "go-socket/core/modules/notification/application/messaging"
	"go-socket/core/shared/config"
)

func BuildMessageHandler(cfg *config.Config, appCtx *appCtx.AppContext) (notificationmessaging.MessageHandler, error) {
	return notificationmessaging.NewMessageHandler(cfg, appCtx.GetSMTP())
}
