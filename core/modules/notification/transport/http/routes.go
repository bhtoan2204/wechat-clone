// CODE_GENERATOR - do not edit: routing
package http

import (
	"wechat-clone/core/modules/notification/application/dto/in"
	"wechat-clone/core/modules/notification/application/dto/out"
	"wechat-clone/core/modules/notification/transport/http/handler"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(_ *gin.RouterGroup) {}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	savePushSubscription cqrs.Dispatcher[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse],
	listNotification cqrs.Dispatcher[*in.ListNotificationRequest, *out.ListNotificationResponse],
	markNotificationRead cqrs.Dispatcher[*in.MarkNotificationReadRequest, *out.MarkNotificationReadResponse],
	markAllNotificationsRead cqrs.Dispatcher[*in.MarkAllNotificationsReadRequest, *out.MarkAllNotificationsReadResponse],
	getUnreadNotificationCount cqrs.Dispatcher[*in.GetUnreadNotificationCountRequest, *out.GetUnreadNotificationCountResponse],
) {
	routes.POST("/notification/push-subscriptions", httpx.Wrap(handler.NewSavePushSubscriptionHandler(savePushSubscription)))
	routes.GET("/notification/list", httpx.Wrap(handler.NewListNotificationHandler(listNotification)))
	routes.PATCH("/notification/:notification_id/read", httpx.Wrap(handler.NewMarkNotificationReadHandler(markNotificationRead)))
	routes.PATCH("/notification/read-all", httpx.Wrap(handler.NewMarkAllNotificationsReadHandler(markAllNotificationsRead)))
	routes.GET("/notification/unread-count", httpx.Wrap(handler.NewGetUnreadNotificationCountHandler(getUnreadNotificationCount)))
}
