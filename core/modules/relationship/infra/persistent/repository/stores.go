package repository

import (
	"context"

	"wechat-clone/core/modules/relationship/domain/entity"
)

type friendshipStore interface {
	ExistsBetween(ctx context.Context, userA, userB string) (bool, error)
	Create(ctx context.Context, friendship *entity.Friendship) error
	DeleteBetween(ctx context.Context, userA, userB string) (bool, error)
}

type followRelationStore interface {
	Exists(ctx context.Context, followerID, followeeID string) (bool, error)
	Create(ctx context.Context, relation *entity.FollowRelation) error
	Delete(ctx context.Context, followerID, followeeID string) (bool, error)
}

type blockRelationStore interface {
	Exists(ctx context.Context, blockerID, blockedID string) (bool, error)
	ExistsAnyDirection(ctx context.Context, userA, userB string) (bool, error)
	Create(ctx context.Context, relation *entity.BlockRelation) error
	Delete(ctx context.Context, blockerID, blockedID string) (bool, error)
}

type userRelationshipCounterStore interface {
	ApplyDeltas(ctx context.Context, deltas map[string]entity.UserRelationshipCounterDelta) error
}

type relationshipAccountStore interface {
	ProjectAccount(ctx context.Context, account *entity.AccountProjection) error
	GetByID(ctx context.Context, accountID string) (*entity.AccountProjection, error)
	Exists(ctx context.Context, accountID string) (bool, error)
}

type relationshipPairGuardStore interface {
	LockPair(ctx context.Context, userA, userB string) error
}
