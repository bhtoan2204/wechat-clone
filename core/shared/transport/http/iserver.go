package http

import (
	"context"

	"github.com/gin-gonic/gin"
)

//go:generate mockgen -package=http -destination=iserver_mock.go -source=iserver.go
type HTTPServer interface {
	RegisterPublicRoutes(routes *gin.RouterGroup)
	RegisterPrivateRoutes(routes *gin.RouterGroup)
	Stop(ctx context.Context) error
}
