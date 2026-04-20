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

type listFriendsHandler struct {
}

func NewListFriends(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.ListFriendsRequest, *out.ListFriendsResponse] {
	return &listFriendsHandler{}
}

func (u *listFriendsHandler) Handle(ctx context.Context, req *in.ListFriendsRequest) (*out.ListFriendsResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
