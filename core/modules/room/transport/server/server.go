package server

import (
	"context"
	"fmt"
	roomcommand "go-socket/core/modules/room/application/command"
	roomquery "go-socket/core/modules/room/application/query"
	roomhttp "go-socket/core/modules/room/transport/http"
	roomsocket "go-socket/core/modules/room/transport/websocket"
	stackerr "go-socket/core/shared/pkg/stackErr"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type roomServer struct {
	commandBus roomcommand.Bus
	queryBus   roomquery.Bus
	roomHub    roomsocket.IHub
}

func NewHTTPServer(commandBus roomcommand.Bus, queryBus roomquery.Bus, roomHub roomsocket.IHub) (infrahttp.HTTPServer, error) {
	if roomHub == nil {
		return nil, stackerr.Error(fmt.Errorf("room hub can not be nil"))
	}

	return &roomServer{
		commandBus: commandBus,
		queryBus:   queryBus,
		roomHub:    roomHub,
	}, nil
}

func (s *roomServer) RegisterPublicRoutes(_ *gin.RouterGroup) {
}

func (s *roomServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	roomhttp.RegisterPrivateRoutes(routes, s.commandBus, s.queryBus, s.roomHub)
}

func (s *roomServer) Stop(ctx context.Context) error {
	s.roomHub.Close(ctx)
	return nil
}
