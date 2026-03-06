package server

import (
	"context"
	accountcommand "go-socket/core/modules/account/application/command"
	accountquery "go-socket/core/modules/account/application/query"
	accounthttp "go-socket/core/modules/account/transport/http"

	"github.com/gin-gonic/gin"
)

type Server interface {
	RegisterPublicRoutes(routes *gin.RouterGroup)
	RegisterPrivateRoutes(routes *gin.RouterGroup)
	Stop(ctx context.Context) error
}

type accountServer struct {
	commandBus accountcommand.Bus
	queryBus   accountquery.Bus
}

func NewServer(commandBus accountcommand.Bus, queryBus accountquery.Bus) (Server, error) {
	return &accountServer{
		commandBus: commandBus,
		queryBus:   queryBus,
	}, nil
}

func (s *accountServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	accounthttp.RegisterPublicRoutes(routes, s.commandBus)
}

func (s *accountServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	accounthttp.RegisterPrivateRoutes(routes, s.commandBus, s.queryBus)
}

func (s *accountServer) Stop(_ context.Context) error {
	return nil
}
