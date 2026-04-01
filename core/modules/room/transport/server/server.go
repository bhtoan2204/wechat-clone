package server

import (
	"context"
	"fmt"
	roomin "go-socket/core/modules/room/application/dto/in"
	roomout "go-socket/core/modules/room/application/dto/out"
	roomhttp "go-socket/core/modules/room/transport/http"
	roomsocket "go-socket/core/modules/room/transport/websocket"
	"go-socket/core/shared/pkg/cqrs"
	stackerr "go-socket/core/shared/pkg/stackErr"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type roomServer struct {
	createRoom cqrs.Dispatcher[*roomin.CreateRoomRequest, *roomout.CreateRoomResponse]
	updateRoom cqrs.Dispatcher[*roomin.UpdateRoomRequest, *roomout.UpdateRoomResponse]
	deleteRoom cqrs.Dispatcher[*roomin.DeleteRoomRequest, *roomout.DeleteRoomResponse]
	getRoom    cqrs.Dispatcher[*roomin.GetRoomRequest, *roomout.GetRoomResponse]
	listRoom   cqrs.Dispatcher[*roomin.ListRoomsRequest, *roomout.ListRoomsResponse]
	roomHub    roomsocket.IHub
}

func NewHTTPServer(
	createRoom cqrs.Dispatcher[*roomin.CreateRoomRequest, *roomout.CreateRoomResponse],
	updateRoom cqrs.Dispatcher[*roomin.UpdateRoomRequest, *roomout.UpdateRoomResponse],
	deleteRoom cqrs.Dispatcher[*roomin.DeleteRoomRequest, *roomout.DeleteRoomResponse],
	getRoom cqrs.Dispatcher[*roomin.GetRoomRequest, *roomout.GetRoomResponse],
	listRoom cqrs.Dispatcher[*roomin.ListRoomsRequest, *roomout.ListRoomsResponse],
	roomHub roomsocket.IHub,
) (infrahttp.HTTPServer, error) {
	if roomHub == nil {
		return nil, stackerr.Error(fmt.Errorf("room hub can not be nil"))
	}

	return &roomServer{
		createRoom: createRoom,
		updateRoom: updateRoom,
		deleteRoom: deleteRoom,
		getRoom:    getRoom,
		listRoom:   listRoom,
		roomHub:    roomHub,
	}, nil
}

func (s *roomServer) RegisterPublicRoutes(_ *gin.RouterGroup) {
}

func (s *roomServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	roomhttp.RegisterPrivateRoutes(routes, s.createRoom, s.updateRoom, s.deleteRoom, s.getRoom, s.listRoom, s.roomHub)
}

func (s *roomServer) Stop(ctx context.Context) error {
	s.roomHub.Close(ctx)
	return nil
}
