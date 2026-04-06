// CODE_GENERATOR: routing
package http

import (
	"go-socket/core/modules/notification/application/dto/in"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(_ *gin.RouterGroup) {}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	savePushSubscription cqrs.Dispatcher[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse],
	listNotification cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse],
) {
	routes.POST("/notification/push-subscriptions", httpx.Wrap(handler.NewSavePushSubscriptionHandler(savePushSubscription)))
	routes.GET("/notification/list", httpx.Wrap(handler.NewListNotificationHandler(listNotification)))
}
