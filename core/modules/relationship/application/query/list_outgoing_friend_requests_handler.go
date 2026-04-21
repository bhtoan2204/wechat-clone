package query

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/relationship/application/dto/in"
	"wechat-clone/core/modules/relationship/application/dto/out"
	relationshipprojection "wechat-clone/core/modules/relationship/application/projection"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type listOutgoingFriendRequestsHandler struct {
	projRepo relationshipprojection.ReadRepository
}

func NewListOutgoingFriendRequests(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
) cqrs.Handler[*in.ListOutgoingFriendRequestsRequest, *out.ListOutgoingFriendRequestsResponse] {
	return &listOutgoingFriendRequestsHandler{projRepo: projRepo}
}

func (u *listOutgoingFriendRequestsHandler) Handle(ctx context.Context, req *in.ListOutgoingFriendRequestsRequest) (*out.ListOutgoingFriendRequestsResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.projRepo.ListOutgoingFriendRequests(ctx, accountID, req.Cursor, normalizeLimit(req.Limit))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if result == nil {
		result = emptyListResult()
	}
	return &out.ListOutgoingFriendRequestsResponse{Items: result.Items, NextCursor: result.NextCursor}, nil
}
