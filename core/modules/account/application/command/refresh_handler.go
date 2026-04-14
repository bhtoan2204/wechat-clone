// CODE_GENERATOR: application-handler
package command

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/service"
	repos "go-socket/core/modules/account/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
)

type refreshHandler struct {
}

func NewRefresh(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	services service.Services,
) cqrs.Handler[*in.RefreshRequest, *out.RefreshResponse] {
	return &refreshHandler{}
}

func (u *refreshHandler) Handle(ctx context.Context, req *in.RefreshRequest) (*out.RefreshResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
