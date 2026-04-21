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

type getRelationshipStatusHandler struct {
	projRepo relationshipprojection.ReadRepository
}

func NewGetRelationshipStatus(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
) cqrs.Handler[*in.GetRelationshipStatusRequest, *out.GetRelationshipStatusResponse] {
	return &getRelationshipStatusHandler{projRepo: projRepo}
}

func (u *getRelationshipStatusHandler) Handle(ctx context.Context, req *in.GetRelationshipStatusRequest) (*out.GetRelationshipStatusResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	pair, err := u.projRepo.GetPair(ctx, accountID, req.TargetUserID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return buildRelationshipStatusResponse(accountID, req.TargetUserID, pair), nil
}
