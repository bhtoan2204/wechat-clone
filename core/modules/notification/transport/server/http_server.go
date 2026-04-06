// CODE_GENERATOR: registry
package server

import (
	"context"

	"go-socket/core/modules/notification/application/dto/in"
	"go-socket/core/modules/notification/application/dto/out"
	notificationhttp "go-socket/core/modules/notification/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type notificationHTTPServer struct {
	savePushSubscription cqrs.Dispatcher[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse]
	listNotification     cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse]
}

func NewHTTPServer(
	savePushSubscription cqrs.Dispatcher[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse],
	listNotification cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse],
) (infrahttp.HTTPServer, error) {
	return &notificationHTTPServer{
		savePushSubscription: savePushSubscription,
		listNotification:     listNotification,
	}, nil
}

func (s *notificationHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	notificationhttp.RegisterPublicRoutes(routes)
}

func (s *notificationHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	notificationhttp.RegisterPrivateRoutes(routes, s.savePushSubscription, s.listNotification)
}

func (s *notificationHTTPServer) Stop(_ context.Context) error {
	return nil
}
