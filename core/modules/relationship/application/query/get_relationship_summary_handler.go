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

type getRelationshipSummaryHandler struct {
	projRepo relationshipprojection.ReadRepository
}

func NewGetRelationshipSummary(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
) cqrs.Handler[*in.GetRelationshipSummaryRequest, *out.GetRelationshipSummaryResponse] {
	return &getRelationshipSummaryHandler{projRepo: projRepo}
}

func (u *getRelationshipSummaryHandler) Handle(ctx context.Context, req *in.GetRelationshipSummaryRequest) (*out.GetRelationshipSummaryResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	pair, err := u.projRepo.GetPair(ctx, accountID, req.TargetUserID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	status := buildRelationshipStatusResponse(accountID, req.TargetUserID, pair)

	friendsCount, err := u.projRepo.CountFriends(ctx, req.TargetUserID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	followersCount, err := u.projRepo.CountFollowers(ctx, req.TargetUserID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	followingCount, err := u.projRepo.CountFollowing(ctx, req.TargetUserID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	mutualFriendsCount, err := u.projRepo.CountMutualFriends(ctx, accountID, req.TargetUserID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.GetRelationshipSummaryResponse{
		FriendsCount:       friendsCount,
		FollowersCount:     followersCount,
		FollowingCount:     followingCount,
		MutualFriendsCount: mutualFriendsCount,
		RelationshipStatus: relationshipStatusLabel(status),
	}, nil
}
