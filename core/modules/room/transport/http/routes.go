package http

import (
	roomin "go-socket/core/modules/room/application/dto/in"
	roomout "go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/transport/http/handler"
	roomsocket "go-socket/core/modules/room/transport/websocket"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	createRoom cqrs.Dispatcher[*roomin.CreateRoomRequest, *roomout.CreateRoomResponse],
	updateRoom cqrs.Dispatcher[*roomin.UpdateRoomRequest, *roomout.UpdateRoomResponse],
	deleteRoom cqrs.Dispatcher[*roomin.DeleteRoomRequest, *roomout.DeleteRoomResponse],
	getRoom cqrs.Dispatcher[*roomin.GetRoomRequest, *roomout.GetRoomResponse],
	listRoom cqrs.Dispatcher[*roomin.ListRoomsRequest, *roomout.ListRoomsResponse],
	roomHub roomsocket.IHub,
) {
	routes.POST("/room/create", httpx.Wrap(handler.NewCreateRoomHandler(createRoom)))
	routes.GET("/room/list", httpx.Wrap(handler.NewListRoomsHandler(listRoom)))
	routes.GET("/room/get", httpx.Wrap(handler.NewGetRoomHandler(getRoom)))
	routes.PUT("/room/update", httpx.Wrap(handler.NewUpdateRoomHandler(updateRoom)))
	routes.DELETE("/room/delete", httpx.Wrap(handler.NewDeleteRoomHandler(deleteRoom)))
	routes.GET("/room/ws", roomsocket.NewWSHandler(roomHub).Handle)
	// routes.POST("/message/create", httpx.Wrap(handler.NewCreateMessageHandler(messageUsecase)))
}
