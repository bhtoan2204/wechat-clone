package command

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/service"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type refreshHandler struct {
	authSvc service.AuthenticationService
}

func NewRefresh(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	services service.Services,
) cqrs.Handler[*in.RefreshRequest, *out.RefreshResponse] {
	return &refreshHandler{
		authSvc: services.AuthenticationService(),
	}
}

func (u *refreshHandler) Handle(ctx context.Context, req *in.RefreshRequest) (*out.RefreshResponse, error) {
	if result, err := u.authSvc.RefreshAuthenticate(ctx, service.RefreshTokenCommand{
		RefreshToken: req.RefreshToken,
		UserAgent:    req.UserAgent,
		IPAddress:    req.IpAddress,
	}); err != nil {
		return nil, stackErr.Error(err)
	} else {
		return &out.RefreshResponse{
			AccessToken:      result.AccessToken,
			RefreshToken:     result.RefreshToken,
			AccessExpiresAt:  result.AccessExpiresAt.UnixMilli(),
			RefreshExpiresAt: result.RefreshExpiresAt.UnixMilli(),
		}, nil
	}
}
