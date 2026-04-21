package repos

import "context"

//go:generate mockgen -package=repos -destination=repos_mock.go -source=repos.go
type Repos interface {
	FriendRequestAggregateRepository() FriendRequestAggregateRepository
	RelationshipPairAggregateRepository() RelationshipPairAggregateRepository
	FriendshipRepository() FriendshipRepository
	FollowRelationRepository() FollowRelationRepository
	BlockRelationRepository() BlockRelationRepository
	UserRelationshipCounterRepository() UserRelationshipCounterRepository
	RelationshipAccountRepository() RelationshipAccountRepository
	RelationshipPairGuardRepository() RelationshipPairGuardRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
