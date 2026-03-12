package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	roomserver "go-socket/core/modules/room/transport/server"
	roomsocket "go-socket/core/modules/room/transport/websocket"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(ctx context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	buses := BuildBuses(appContext)
	roomHub := roomsocket.NewHub(ctx, appContext)

	server, err := roomserver.NewHTTPServer(buses.Command, buses.Query, roomHub)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	return server, nil
}
