package server

import (
	"context"
	notificationin "go-socket/core/modules/notification/application/dto/in"
	notificationout "go-socket/core/modules/notification/application/dto/out"
	notificationhttp "go-socket/core/modules/notification/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type notificationHTTPServer struct {
	savePushSubscription cqrs.Dispatcher[*notificationin.SavePushSubscriptionRequest, *notificationout.SavePushSubscriptionResponse]
	listNotification     cqrs.Dispatcher[*notificationin.ListNotificationRequest, *notificationout.ListNotificationResponse]
}

func NewHTTPServer(
	savePushSubscription cqrs.Dispatcher[*notificationin.SavePushSubscriptionRequest, *notificationout.SavePushSubscriptionResponse],
	listNotification cqrs.Dispatcher[*notificationin.ListNotificationRequest, *notificationout.ListNotificationResponse],
) (infrahttp.HTTPServer, error) {
	return &notificationHTTPServer{
		savePushSubscription: savePushSubscription,
		listNotification:     listNotification,
	}, nil
}

func (s *notificationHTTPServer) RegisterPublicRoutes(_ *gin.RouterGroup) {}

func (s *notificationHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	notificationhttp.RegisterPrivateRoutes(routes, s.savePushSubscription, s.listNotification)
}

func (s *notificationHTTPServer) Stop(_ context.Context) error {
	return nil
}
