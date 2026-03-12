package http

import (
	"context"

	"github.com/gin-gonic/gin"
)

type HTTPServer interface {
	RegisterPublicRoutes(routes *gin.RouterGroup)
	RegisterPrivateRoutes(routes *gin.RouterGroup)
	Stop(ctx context.Context) error
}
