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

type listIncomingFriendRequestsHandler struct {
	projRepo relationshipprojection.ReadRepository
}

func NewListIncomingFriendRequests(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
) cqrs.Handler[*in.ListIncomingFriendRequestsRequest, *out.ListIncomingFriendRequestsResponse] {
	return &listIncomingFriendRequestsHandler{projRepo: projRepo}
}

func (u *listIncomingFriendRequestsHandler) Handle(ctx context.Context, req *in.ListIncomingFriendRequestsRequest) (*out.ListIncomingFriendRequestsResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.projRepo.ListIncomingFriendRequests(ctx, accountID, req.Cursor, normalizeLimit(req.Limit))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if result == nil {
		result = emptyListResult()
	}
	return &out.ListIncomingFriendRequestsResponse{Items: result.Items, NextCursor: result.NextCursor}, nil
}
