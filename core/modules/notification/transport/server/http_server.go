// CODE_GENERATOR: registry
package server

import (
	"context"

	"wechat-clone/core/modules/notification/application/dto/in"
	"wechat-clone/core/modules/notification/application/dto/out"
	notificationhttp "wechat-clone/core/modules/notification/transport/http"
	notificationsocket "wechat-clone/core/modules/notification/transport/websocket"
	"wechat-clone/core/shared/pkg/cqrs"
	infrahttp "wechat-clone/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type notificationHTTPServer struct {
	savePushSubscription       cqrs.Dispatcher[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse]
	listNotification           cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse]
	markNotificationRead       cqrs.Dispatcher[*in.MarkNotificationReadRequest, *out.MarkNotificationReadResponse]
	markAllNotificationsRead   cqrs.Dispatcher[*in.MarkAllNotificationsReadRequest, *out.MarkAllNotificationsReadResponse]
	getUnreadNotificationCount cqrs.Dispatcher[*in.GetUnreadNotificationCountRequest, *out.GetUnreadNotificationCountResponse]
	socketHandler              gin.HandlerFunc
	socketStopper              func(context.Context)
}

func NewHTTPServer(
	savePushSubscription cqrs.Dispatcher[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse],
	listNotification cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse],
	markNotificationRead cqrs.Dispatcher[*in.MarkNotificationReadRequest, *out.MarkNotificationReadResponse],
	markAllNotificationsRead cqrs.Dispatcher[*in.MarkAllNotificationsReadRequest, *out.MarkAllNotificationsReadResponse],
	getUnreadNotificationCount cqrs.Dispatcher[*in.GetUnreadNotificationCountRequest, *out.GetUnreadNotificationCountResponse],
	socketHandler gin.HandlerFunc,
	socketStopper func(context.Context),
) (infrahttp.HTTPServer, error) {
	return &notificationHTTPServer{
		savePushSubscription:       savePushSubscription,
		listNotification:           listNotification,
		markNotificationRead:       markNotificationRead,
		markAllNotificationsRead:   markAllNotificationsRead,
		getUnreadNotificationCount: getUnreadNotificationCount,
		socketHandler:              socketHandler,
		socketStopper:              socketStopper,
	}, nil
}

func (s *notificationHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	notificationhttp.RegisterPublicRoutes(routes)
}

func (s *notificationHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	notificationhttp.RegisterPrivateRoutes(routes, s.savePushSubscription, s.listNotification, s.markNotificationRead, s.markAllNotificationsRead, s.getUnreadNotificationCount)
}

func (s *notificationHTTPServer) RegisterSocketRoutes(routes *gin.RouterGroup) {
	notificationsocket.RegisterPrivateRoutes(routes, s.socketHandler)
}

func (s *notificationHTTPServer) Stop(ctx context.Context) error {
	if s.socketStopper != nil {
		s.socketStopper(ctx)
	}
	return nil
}
