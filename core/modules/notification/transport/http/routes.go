package http

import (
	notificationin "go-socket/core/modules/notification/application/dto/in"
	notificationout "go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	savePushSubscription cqrs.Dispatcher[*notificationin.SavePushSubscriptionRequest, *notificationout.SavePushSubscriptionResponse],
	listNotification cqrs.Dispatcher[*notificationin.ListNotificationRequest, *notificationout.ListNotificationResponse],
) {
	routes.POST("/notification/push-subscriptions", httpx.Wrap(handler.NewSavePushSubscriptionHandler(savePushSubscription)))
	routes.GET("/notification/list", httpx.Wrap(handler.NewListNotificationHandler(listNotification)))
}
