package command

import (
	"context"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/service"
	"go-socket/core/modules/account/application/support"
	repos "go-socket/core/modules/account/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type logoutHandler struct {
	authService service.AuthenticationService
}

func NewLogoutHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos, services service.Services) cqrs.Handler[*in.LogoutRequest, *out.LogoutResponse] {
	return &logoutHandler{
		authService: services.AuthenticationService(),
	}
}

func (u *logoutHandler) Handle(ctx context.Context, req *in.LogoutRequest) (*out.LogoutResponse, error) {
	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := u.authService.Logout(ctx, service.LogoutCommand{
		AccountID:    accountID,
		RefreshToken: req.Token,
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.LogoutResponse{
		Message: "Logout successful",
	}, nil
}
