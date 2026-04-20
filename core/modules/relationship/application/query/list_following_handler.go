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

type listFollowingHandler struct {
}

func NewListFollowing(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.ListFollowingRequest, *out.ListFollowingResponse] {
	return &listFollowingHandler{}
}

func (u *listFollowingHandler) Handle(ctx context.Context, req *in.ListFollowingRequest) (*out.ListFollowingResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
