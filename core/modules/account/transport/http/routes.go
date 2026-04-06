// CODE_GENERATOR: routing
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
	confirmVerifyEmail cqrs.Dispatcher[*in.ConfirmVerifyEmailRequest, *out.ConfirmVerifyEmailResponse],
) {
	routes.POST("/auth/login", httpx.Wrap(handler.NewLoginHandler(login)))
	routes.POST("/auth/register", httpx.Wrap(handler.NewRegisterHandler(register)))
	routes.POST("/account/verify-email/confirm", httpx.Wrap(handler.NewConfirmVerifyEmailHandler(confirmVerifyEmail)))
}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	logout cqrs.Dispatcher[*in.LogoutRequest, *out.LogoutResponse],
	getProfile cqrs.Dispatcher[*in.GetProfileRequest, *out.GetProfileResponse],
	updateProfile cqrs.Dispatcher[*in.UpdateProfileRequest, *out.UpdateProfileResponse],
	verifyEmail cqrs.Dispatcher[*in.VerifyEmailRequest, *out.VerifyEmailResponse],
	changePassword cqrs.Dispatcher[*in.ChangePasswordRequest, *out.ChangePasswordResponse],
	getAvatar cqrs.Dispatcher[*in.GetAvatarRequest, *out.GetAvatarResponse],
	createPresignedUrl cqrs.Dispatcher[*in.CreatePresignedUrlRequest, *out.CreatePresignedUrlResponse],
) {
	routes.POST("/auth/logout", httpx.Wrap(handler.NewLogoutHandler(logout)))
	routes.GET("/account/profile", httpx.Wrap(handler.NewGetProfileHandler(getProfile)))
	routes.PUT("/account/profile", httpx.Wrap(handler.NewUpdateProfileHandler(updateProfile)))
	routes.POST("/account/verify-email", httpx.Wrap(handler.NewVerifyEmailHandler(verifyEmail)))
	routes.PUT("/account/change-password", httpx.Wrap(handler.NewChangePasswordHandler(changePassword)))
	routes.GET("/account/avatar/:account_id", httpx.Wrap(handler.NewGetAvatarHandler(getAvatar)))
	routes.POST("/account/avatar/presigned-url", httpx.Wrap(handler.NewCreatePresignedUrlHandler(createPresignedUrl)))
}
