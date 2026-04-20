package repos

import (
	"context"
	"wechat-clone/core/modules/relationship/domain/entity"
)

type BlockRelationRepository interface {
	Exists(ctx context.Context, blockerID, blockedID string) (bool, error)
	ExistsAnyDirection(ctx context.Context, userA, userB string) (bool, error)
	Create(ctx context.Context, relation *entity.BlockRelation) error
	Delete(ctx context.Context, blockerID, blockedID string) (bool, error)
}
