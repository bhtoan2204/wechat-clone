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

type listOutgoingFriendRequestsHandler struct {
}

func NewListOutgoingFriendRequests(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.ListOutgoingFriendRequestsRequest, *out.ListOutgoingFriendRequestsResponse] {
	return &listOutgoingFriendRequestsHandler{}
}

func (u *listOutgoingFriendRequestsHandler) Handle(ctx context.Context, req *in.ListOutgoingFriendRequestsRequest) (*out.ListOutgoingFriendRequestsResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
