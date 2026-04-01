package http

import (
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(
	routes *gin.RouterGroup,
	login cqrs.Dispatcher[*in.LoginRequest, *out.LoginResponse],
	register cqrs.Dispatcher[*in.RegisterRequest, *out.RegisterResponse],
) {
	routes.POST("/auth/login", httpx.Wrap(handler.NewLoginHandler(login)))
	routes.POST("/auth/register", httpx.Wrap(handler.NewRegisterHandler(register)))
}

func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	logout cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse],
	getProfile cqrs.Dispatcher[*in.GetProfileRequest, *out.GetProfileResponse],
) {
	routes.POST("/auth/logout", httpx.Wrap(handler.NewLogoutHandler(logout)))
	routes.GET("/auth/profile", httpx.Wrap(handler.NewGetProfileHandler(getProfile)))
}
