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

type getMutualFriendsHandler struct {
}

func NewGetMutualFriends(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.GetMutualFriendsRequest, *out.GetMutualFriendsResponse] {
	return &getMutualFriendsHandler{}
}

func (u *getMutualFriendsHandler) Handle(ctx context.Context, req *in.GetMutualFriendsRequest) (*out.GetMutualFriendsResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
