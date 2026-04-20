package repos

import (
	"context"
	"wechat-clone/core/modules/relationship/domain/entity"
)

type FollowRelationRepository interface {
	Exists(ctx context.Context, followerID, followeeID string) (bool, error)
	Create(ctx context.Context, relation *entity.FollowRelation) error
	Delete(ctx context.Context, followerID, followeeID string) (bool, error)
}
