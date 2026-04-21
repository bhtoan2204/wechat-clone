package projection

import "context"

type ReadRepository interface {
	GetPair(ctx context.Context, userA, userB string) (*RelationshipPairProjection, error)
	SavePair(ctx context.Context, projection *RelationshipPairProjection) error
	ListFriends(ctx context.Context, userID, cursor string, limit int) (*RelationshipListResult, error)
	ListFollowers(ctx context.Context, userID, cursor string, limit int) (*RelationshipListResult, error)
	ListFollowing(ctx context.Context, userID, cursor string, limit int) (*RelationshipListResult, error)
	ListBlockedUsers(ctx context.Context, userID, cursor string, limit int) (*RelationshipListResult, error)
	ListIncomingFriendRequests(ctx context.Context, userID, cursor string, limit int) (*RelationshipListResult, error)
	ListOutgoingFriendRequests(ctx context.Context, userID, cursor string, limit int) (*RelationshipListResult, error)
	ListMutualFriends(ctx context.Context, userID, targetUserID, cursor string, limit int) (*RelationshipListResult, error)
	CountFriends(ctx context.Context, userID string) (int64, error)
	CountFollowers(ctx context.Context, userID string) (int64, error)
	CountFollowing(ctx context.Context, userID string) (int64, error)
	CountMutualFriends(ctx context.Context, userID, targetUserID string) (int64, error)
}
