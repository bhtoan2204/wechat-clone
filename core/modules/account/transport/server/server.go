// CODE_GENERATOR: registry
package server

import (
	"context"

	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	accounthttp "go-socket/core/modules/account/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type accountHTTPServer struct {
	login              cqrs.Dispatcher[*in.LoginRequest, *out.LoginResponse]
	register           cqrs.Dispatcher[*in.RegisterRequest, *out.RegisterResponse]
	logout             cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse]
	getProfile         cqrs.Dispatcher[*in.GetProfileRequest, *out.GetProfileResponse]
	updateProfile      cqrs.Dispatcher[*in.UpdateProfileRequest, *out.UpdateProfileResponse]
	verifyEmail        cqrs.Dispatcher[*in.VerifyEmailRequest, *out.VerifyEmailResponse]
	confirmVerifyEmail cqrs.Dispatcher[*in.ConfirmVerifyEmailRequest, *out.ConfirmVerifyEmailResponse]
	changePassword     cqrs.Dispatcher[*in.ChangePasswordRequest, *out.ChangePasswordResponse]
	getAvatar          cqrs.Dispatcher[*in.GetAvatarRequest, *out.GetAvatarResponse]
	createPresignedUrl cqrs.Dispatcher[*in.CreatePresignedUrlRequest, *out.CreatePresignedUrlResponse]
}

func NewHTTPServer(
	login cqrs.Dispatcher[*in.LoginRequest, *out.LoginResponse],
	register cqrs.Dispatcher[*in.RegisterRequest, *out.RegisterResponse],
	logout cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse],
	getProfile cqrs.Dispatcher[*in.GetProfileRequest, *out.GetProfileResponse],
	updateProfile cqrs.Dispatcher[*in.UpdateProfileRequest, *out.UpdateProfileResponse],
	verifyEmail cqrs.Dispatcher[*in.VerifyEmailRequest, *out.VerifyEmailResponse],
	confirmVerifyEmail cqrs.Dispatcher[*in.ConfirmVerifyEmailRequest, *out.ConfirmVerifyEmailResponse],
	changePassword cqrs.Dispatcher[*in.ChangePasswordRequest, *out.ChangePasswordResponse],
	getAvatar cqrs.Dispatcher[*in.GetAvatarRequest, *out.GetAvatarResponse],
	createPresignedUrl cqrs.Dispatcher[*in.CreatePresignedUrlRequest, *out.CreatePresignedUrlResponse],
) (infrahttp.HTTPServer, error) {
	return &accountHTTPServer{
		login:              login,
		register:           register,
		logout:             logout,
		getProfile:         getProfile,
		updateProfile:      updateProfile,
		verifyEmail:        verifyEmail,
		confirmVerifyEmail: confirmVerifyEmail,
		changePassword:     changePassword,
		getAvatar:          getAvatar,
		createPresignedUrl: createPresignedUrl,
	}, nil
}

func (s *accountHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	accounthttp.RegisterPublicRoutes(routes, s.login, s.register, s.confirmVerifyEmail)
}

func (s *accountHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	accounthttp.RegisterPrivateRoutes(routes, s.logout, s.getProfile, s.updateProfile, s.verifyEmail, s.changePassword, s.getAvatar, s.createPresignedUrl)
}

func (s *accountHTTPServer) Stop(_ context.Context) error {
	return nil
}
