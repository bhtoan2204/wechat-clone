package server

import (
	"context"
	notificationcommand "go-socket/core/modules/notification/application/command"
	notificationquery "go-socket/core/modules/notification/application/query"
	notificationhttp "go-socket/core/modules/notification/transport/http"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type notificationHTTPServer struct {
	commandBus notificationcommand.Bus
	queryBus   notificationquery.Bus
}

func NewHTTPServer(commandBus notificationcommand.Bus, queryBus notificationquery.Bus) (infrahttp.HTTPServer, error) {
	return &notificationHTTPServer{commandBus: commandBus, queryBus: queryBus}, nil
}

func (s *notificationHTTPServer) RegisterPublicRoutes(_ *gin.RouterGroup) {}

func (s *notificationHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	notificationhttp.RegisterPrivateRoutes(routes, s.commandBus, s.queryBus)
}

func (s *notificationHTTPServer) Stop(_ context.Context) error {
	return nil
}
