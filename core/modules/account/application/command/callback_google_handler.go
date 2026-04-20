package command

import (
	"context"
	"errors"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/provider"
	"wechat-clone/core/modules/account/application/service"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type callbackGoogleHandler struct {
	authProviderRegistry *provider.AuthProviderRegistry
	services             service.Services
}

func NewCallbackGoogle(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	services service.Services,
	authProviderRegistry *provider.AuthProviderRegistry,
) cqrs.Handler[*in.CallbackGoogleRequest, *out.CallbackGoogleResponse] {
	return &callbackGoogleHandler{
		authProviderRegistry: authProviderRegistry,
		services:             services,
	}
}

func (u *callbackGoogleHandler) Handle(ctx context.Context, req *in.CallbackGoogleRequest) (*out.CallbackGoogleResponse, error) {
	log := logging.FromContext(ctx)
	googleProvider, err := u.authProviderRegistry.Get("google")
	if err != nil {
		return nil, stackErr.Error(err)
	}
	callbackData, err := googleProvider.Callback(ctx, req.Code)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	userInfo, err := googleProvider.UserInfo(ctx, callbackData.AccessToken)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.services.AuthenticationService().OpenAuthenticate(ctx, service.OpenAuthenticateAccountCommand{
		UserInfo: *userInfo,
		Device: service.DeviceCommand{
			DeviceUID:  req.DeviceUid,
			DeviceName: req.DeviceName,
			DeviceType: req.DeviceType,
			OSName:     req.OsName,
			OSVersion:  req.OsVersion,
			AppVersion: req.AppVersion,
			UserAgent:  req.UserAgent,
			IPAddress:  req.IpAddress,
		},
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAuthenticationAccountNotFound):
			log.Errorw("Account not found", zap.Any("userInfo", userInfo))
			return nil, stackErr.Error(ErrAccountNotFound)
		case errors.Is(err, service.ErrAuthenticationInvalidPassword):
			log.Errorw("Invalid credentials", zap.Any("userInfo", userInfo))
			return nil, stackErr.Error(ErrInvalidCredentials)
		default:
			log.Errorw("Login failed", zap.Error(err), zap.Any("userInfo", userInfo))
			return nil, stackErr.Error(err)
		}
	}

	return &out.CallbackGoogleResponse{
		AccessToken:      result.AccessToken,
		AccessExpiresAt:  result.AccessExpiresAt.UnixMilli(),
		RefreshToken:     result.RefreshToken,
		RefreshExpiresAt: result.RefreshExpiresAt.UnixMilli(),
	}, nil
}
