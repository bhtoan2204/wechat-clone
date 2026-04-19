package assembly

import (
	"context"

	appCtx "wechat-clone/core/context"
	notificationcommand "wechat-clone/core/modules/notification/application/command"
	notificationquery "wechat-clone/core/modules/notification/application/query"
	notificationservice "wechat-clone/core/modules/notification/application/service"
	notificationrepo "wechat-clone/core/modules/notification/infra/persistent/repository"
	notificationserver "wechat-clone/core/modules/notification/transport/server"
	notificationsocket "wechat-clone/core/modules/notification/transport/websocket"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/transport/http"
	sharedsocket "wechat-clone/core/shared/transport/websocket"

	"github.com/gin-gonic/gin"
)

func buildHTTPServer(_ context.Context, appCtx *appCtx.AppContext) (http.HTTPServer, error) {
	notificationRepos, err := notificationrepo.NewRepoImpl(appCtx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	notificationReadRepo, err := notificationrepo.NewNotificationReadRepository(appCtx.GetCassandraSession())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	realtimeService := notificationservice.NewRealtimeService(appCtx)
	savePushSubscription := cqrs.NewDispatcher(notificationcommand.NewSavePushSubscriptionHandler(notificationRepos))
	listNotification := cqrs.NewDispatcher(notificationquery.NewListNotificationHandler(notificationReadRepo))
	markNotificationRead := cqrs.NewDispatcher(notificationcommand.NewMarkNotificationReadHandler(notificationRepos, realtimeService))
	markAllNotificationsRead := cqrs.NewDispatcher(notificationcommand.NewMarkAllNotificationsReadHandler(notificationRepos, realtimeService))
	getUnreadNotificationCount := cqrs.NewDispatcher(notificationquery.NewGetUnreadNotificationCountHandler(notificationReadRepo))

	socketHub := notificationsocket.NewHub(appCtx)
	socketHandler := notificationsocket.NewWSHandler(appCtx, socketHub, sharedsocket.NewUpgrader())

	var socketHandleFn func(*gin.Context)
	var socketStopFn func(context.Context)
	if socketHandler != nil {
		socketHandleFn = socketHandler.Handle
		socketStopFn = socketHandler.Close
	}

	server, err := notificationserver.NewHTTPServer(
		savePushSubscription,
		listNotification,
		markNotificationRead,
		markAllNotificationsRead,
		getUnreadNotificationCount,
		socketHandleFn,
		socketStopFn,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
