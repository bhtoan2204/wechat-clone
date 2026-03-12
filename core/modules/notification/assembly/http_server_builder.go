package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	notificationcommand "go-socket/core/modules/notification/application/command"
	notificationquery "go-socket/core/modules/notification/application/query"
	notificationrepo "go-socket/core/modules/notification/infra/persistent/repository"
	notificationserver "go-socket/core/modules/notification/transport/server"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(_ context.Context, appCtx *appCtx.AppContext) (http.HTTPServer, error) {
	notificationRepos := notificationrepo.NewRepoImpl(appCtx)
	savePushSubscriptionHandler := notificationcommand.NewSavePushSubscriptionHandler(notificationRepos)
	commandBus := notificationcommand.NewBus(savePushSubscriptionHandler)

	listNotificationHandler := notificationquery.NewListNotificationHandler(notificationRepos)
	queryBus := notificationquery.NewBus(listNotificationHandler)
	server, err := notificationserver.NewHTTPServer(commandBus, queryBus)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	return server, nil
}
