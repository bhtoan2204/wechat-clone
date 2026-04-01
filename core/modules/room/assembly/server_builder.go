package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	roomcommand "go-socket/core/modules/room/application/command"
	roomquery "go-socket/core/modules/room/application/query"
	roomrepo "go-socket/core/modules/room/infra/persistent/repository"
	roomserver "go-socket/core/modules/room/transport/server"
	roomsocket "go-socket/core/modules/room/transport/websocket"
	"go-socket/core/shared/pkg/cqrs"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(ctx context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	roomRepos := roomrepo.NewRepoImpl(appContext)
	createRoom := cqrs.NewDispatcher(roomcommand.NewCreateRoomHandler(roomRepos))
	updateRoom := cqrs.NewDispatcher(roomcommand.NewUpdateRoomHandler(roomRepos.RoomRepository()))
	deleteRoom := cqrs.NewDispatcher(roomcommand.NewDeleteRoomHandler(roomRepos.RoomRepository()))
	getRoom := cqrs.NewDispatcher(roomquery.NewGetRoomHandler(roomRepos.RoomRepository()))
	listRoom := cqrs.NewDispatcher(roomquery.NewListRoomHandler(roomRepos.RoomRepository()))
	roomHub := roomsocket.NewHub(ctx, appContext)

	server, err := roomserver.NewHTTPServer(createRoom, updateRoom, deleteRoom, getRoom, listRoom, roomHub)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	return server, nil
}
