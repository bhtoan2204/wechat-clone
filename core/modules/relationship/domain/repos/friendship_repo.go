package repos

import (
	"context"
	"wechat-clone/core/modules/relationship/domain/entity"
)

type FriendshipRepository interface {
	ExistsBetween(ctx context.Context, userA, userB string) (bool, error)
	Create(ctx context.Context, friendship *entity.Friendship) error
	DeleteBetween(ctx context.Context, userA, userB string) (bool, error)
}
