package command

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/provider"
	"wechat-clone/core/modules/account/application/service"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type loginGoogleHandler struct {
	authProviderRegistry *provider.AuthProviderRegistry
}

func NewLoginGoogle(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	services service.Services,
	authProviderRegistry *provider.AuthProviderRegistry,
) cqrs.Handler[*in.LoginGoogleRequest, *out.LoginGoogleResponse] {
	return &loginGoogleHandler{
		authProviderRegistry: authProviderRegistry,
	}
}

func (u *loginGoogleHandler) Handle(ctx context.Context, req *in.LoginGoogleRequest) (*out.LoginGoogleResponse, error) {
	googleProvider, err := u.authProviderRegistry.Get("google")
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &out.LoginGoogleResponse{
		RedirectURL: googleProvider.Login(),
	}, nil
}
