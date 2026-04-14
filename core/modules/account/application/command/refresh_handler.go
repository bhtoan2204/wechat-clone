// CODE_GENERATOR: application-handler
package command

import (
	"context"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/service"
	repos "go-socket/core/modules/account/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
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
