package repos

import (
	"context"

	"wechat-clone/core/modules/relationship/domain/entity"
)

type UserRelationshipCounterRepository interface {
	ApplyDeltas(ctx context.Context, deltas map[string]entity.UserRelationshipCounterDelta) error
}
