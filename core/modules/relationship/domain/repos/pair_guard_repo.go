package repos

import "context"

type RelationshipPairGuardRepository interface {
	LockPair(ctx context.Context, userA, userB string) error
}
