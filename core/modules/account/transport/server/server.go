package server

import (
	"context"
	accountin "go-socket/core/modules/account/application/dto/in"
	accountout "go-socket/core/modules/account/application/dto/out"
	accounthttp "go-socket/core/modules/account/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type accountServer struct {
	login      cqrs.Dispatcher[*accountin.LoginRequest, *accountout.LoginResponse]
	register   cqrs.Dispatcher[*accountin.RegisterRequest, *accountout.RegisterResponse]
	logout     cqrs.Dispatcher[*accountin.LogoutRequest, *accountout.LogoutResponse]
	getProfile cqrs.Dispatcher[*accountin.GetProfileRequest, *accountout.GetProfileResponse]
}

func NewServer(
	login cqrs.Dispatcher[*accountin.LoginRequest, *accountout.LoginResponse],
	register cqrs.Dispatcher[*accountin.RegisterRequest, *accountout.RegisterResponse],
	logout cqrs.Dispatcher[*accountin.LogoutRequest, *accountout.LogoutResponse],
	getProfile cqrs.Dispatcher[*accountin.GetProfileRequest, *accountout.GetProfileResponse],
) (http.HTTPServer, error) {
	return &accountServer{
		login:      login,
		register:   register,
		logout:     logout,
		getProfile: getProfile,
	}, nil
}

func (s *accountServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	accounthttp.RegisterPublicRoutes(routes, s.login, s.register)
}

func (s *accountServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	accounthttp.RegisterPrivateRoutes(routes, s.logout, s.getProfile)
}

func (s *accountServer) Stop(_ context.Context) error {
	return nil
}
