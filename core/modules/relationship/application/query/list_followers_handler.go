package query

import (
	"context"
	"fmt"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/relationship/application/dto/in"
	"wechat-clone/core/modules/relationship/application/dto/out"
	repos "wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
)

type listFollowersHandler struct {
}

func NewListFollowers(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.ListFollowersRequest, *out.ListFollowersResponse] {
	return &listFollowersHandler{}
}

func (u *listFollowersHandler) Handle(ctx context.Context, req *in.ListFollowersRequest) (*out.ListFollowersResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
