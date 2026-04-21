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

type getMutualFriendsHandler struct {
	projRepo relationshipprojection.ReadRepository
}

func NewGetMutualFriends(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
) cqrs.Handler[*in.GetMutualFriendsRequest, *out.GetMutualFriendsResponse] {
	return &getMutualFriendsHandler{projRepo: projRepo}
}

func (u *getMutualFriendsHandler) Handle(ctx context.Context, req *in.GetMutualFriendsRequest) (*out.GetMutualFriendsResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.projRepo.ListMutualFriends(ctx, accountID, req.TargetUserID, req.Cursor, normalizeLimit(req.Limit))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if result == nil {
		result = emptyListResult()
	}
	return &out.GetMutualFriendsResponse{Items: result.Items, NextCursor: result.NextCursor, Total: result.Total}, nil
}
