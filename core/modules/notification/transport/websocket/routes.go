package socket

import "github.com/gin-gonic/gin"

func RegisterPrivateRoutes(routes *gin.RouterGroup, socketHandler gin.HandlerFunc) {
	if socketHandler == nil {
		return
	}

	routes.GET("/notification/socket", socketHandler)
}
