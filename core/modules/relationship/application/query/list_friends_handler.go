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

type listFriendsHandler struct {
	projRepo    relationshipprojection.ReadRepository
	accountRepo AccountReadRepository
}

func NewListFriends(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
	accountRepo AccountReadRepository,
) cqrs.Handler[*in.ListFriendsRequest, *out.ListFriendsResponse] {
	return &listFriendsHandler{projRepo: projRepo, accountRepo: accountRepo}
}

func (u *listFriendsHandler) Handle(ctx context.Context, req *in.ListFriendsRequest) (*out.ListFriendsResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.projRepo.ListFriends(ctx, normalizeListTarget(accountID, req.UserID), req.Cursor, normalizeLimit(req.Limit))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if result == nil {
		result = emptyListResult()
	}
	items, err := mapRelationshipAccountSummaries(ctx, u.accountRepo, result.Items)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &out.ListFriendsResponse{Items: items, NextCursor: result.NextCursor, Total: result.Total}, nil
}
