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

type listIncomingFriendRequestsHandler struct {
}

func NewListIncomingFriendRequests(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.ListIncomingFriendRequestsRequest, *out.ListIncomingFriendRequestsResponse] {
	return &listIncomingFriendRequestsHandler{}
}

func (u *listIncomingFriendRequestsHandler) Handle(ctx context.Context, req *in.ListIncomingFriendRequestsRequest) (*out.ListIncomingFriendRequestsResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
