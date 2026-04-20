package aggregate

import (
	"strings"

	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/modules/relationship/domain/policy"
)

type RelationshipPairSnapshot struct {
	ActorID          string
	TargetID         string
	TargetExists     bool
	FriendRequest    *FriendRequestAggregate
	Friendship       *entity.Friendship
	Following        *entity.FollowRelation
	FollowedBy       *entity.FollowRelation
	Blocking         *entity.BlockRelation
	BlockedBy        *entity.BlockRelation
	AggregateVersion int
}

type RelationshipChangeKind string

const (
	RelationshipChangeNone   RelationshipChangeKind = ""
	RelationshipChangeUpsert RelationshipChangeKind = "UPSERT"
	RelationshipChangeDelete RelationshipChangeKind = "DELETE"
)

type FriendRequestPersistenceIntent struct {
	Kind      RelationshipChangeKind
	Aggregate *FriendRequestAggregate
}

type FriendshipPersistenceIntent struct {
	Kind  RelationshipChangeKind
	Value *entity.Friendship
}

type FollowPersistenceIntent struct {
	Kind       RelationshipChangeKind
	Value      *entity.FollowRelation
	FollowerID string
	FolloweeID string
}

type BlockPersistenceIntent struct {
	Kind      RelationshipChangeKind
	Value     *entity.BlockRelation
	BlockerID string
	BlockedID string
}

type RelationshipPairChanges struct {
	FriendRequest FriendRequestPersistenceIntent
	Friendship    FriendshipPersistenceIntent
	Following     FollowPersistenceIntent
	FollowedBy    FollowPersistenceIntent
	Block         BlockPersistenceIntent
	CounterDeltas map[string]entity.UserRelationshipCounterDelta
}

type relationshipPairState struct {
	actorID      string
	targetID     string
	targetExists bool

	friendRequest *FriendRequestAggregate
	friendship    *entity.Friendship
	following     *entity.FollowRelation
	followedBy    *entity.FollowRelation
	blocking      *entity.BlockRelation
	blockedBy     *entity.BlockRelation
}

func newRelationshipPairState(snapshot RelationshipPairSnapshot) relationshipPairState {
	return relationshipPairState{
		actorID:       snapshot.ActorID,
		targetID:      snapshot.TargetID,
		targetExists:  snapshot.TargetExists,
		friendRequest: snapshot.FriendRequest,
		friendship:    snapshot.Friendship,
		following:     snapshot.Following,
		followedBy:    snapshot.FollowedBy,
		blocking:      snapshot.Blocking,
		blockedBy:     snapshot.BlockedBy,
	}
}

func newRelationshipPairChanges() RelationshipPairChanges {
	return RelationshipPairChanges{
		CounterDeltas: map[string]entity.UserRelationshipCounterDelta{},
	}
}

func (s relationshipPairState) toPolicyState() policy.PairState {
	return policy.PairState{
		ActorID:             s.actorID,
		TargetID:            s.targetID,
		TargetExists:        s.targetExists,
		IsFriend:            s.friendship != nil,
		IsFollowing:         s.following != nil,
		HasOutgoingRequest:  s.hasPendingOutgoingRequest(),
		HasIncomingRequest:  s.hasPendingIncomingRequest(),
		HasBlockingRelation: s.blocking != nil || s.blockedBy != nil,
		HasBlockedTarget:    s.blocking != nil,
	}
}

func (s relationshipPairState) hasPendingOutgoingRequest() bool {
	return s.friendRequest != nil &&
		s.friendRequest.FriendRequest != nil &&
		s.friendRequest.FriendRequest.IsPending() &&
		s.friendRequest.FriendRequest.RequesterID == s.actorID &&
		s.friendRequest.FriendRequest.AddresseeID == s.targetID
}

func (s relationshipPairState) hasPendingIncomingRequest() bool {
	return s.friendRequest != nil &&
		s.friendRequest.FriendRequest != nil &&
		s.friendRequest.FriendRequest.IsPending() &&
		s.friendRequest.FriendRequest.RequesterID == s.targetID &&
		s.friendRequest.FriendRequest.AddresseeID == s.actorID
}

func (c *RelationshipPairChanges) trackFriendRequest(agg *FriendRequestAggregate) {
	if agg == nil {
		return
	}
	c.FriendRequest = FriendRequestPersistenceIntent{
		Kind:      RelationshipChangeUpsert,
		Aggregate: agg,
	}
}

func (c *RelationshipPairChanges) createFriendship(friendship *entity.Friendship) {
	if friendship == nil {
		return
	}
	c.Friendship = FriendshipPersistenceIntent{
		Kind:  RelationshipChangeUpsert,
		Value: friendship,
	}
}

func (c *RelationshipPairChanges) deleteFriendship(friendship *entity.Friendship) {
	if friendship == nil {
		return
	}
	c.Friendship = FriendshipPersistenceIntent{
		Kind:  RelationshipChangeDelete,
		Value: friendship,
	}
}

func (c *RelationshipPairChanges) createFollowing(relation *entity.FollowRelation) {
	if relation == nil {
		return
	}
	c.Following = FollowPersistenceIntent{
		Kind:       RelationshipChangeUpsert,
		Value:      relation,
		FollowerID: relation.FollowerID,
		FolloweeID: relation.FolloweeID,
	}
}

func (c *RelationshipPairChanges) deleteFollowing(relation *entity.FollowRelation) {
	if relation == nil {
		return
	}
	c.Following = FollowPersistenceIntent{
		Kind:       RelationshipChangeDelete,
		Value:      relation,
		FollowerID: relation.FollowerID,
		FolloweeID: relation.FolloweeID,
	}
}

func (c *RelationshipPairChanges) deleteFollowedBy(relation *entity.FollowRelation) {
	if relation == nil {
		return
	}
	c.FollowedBy = FollowPersistenceIntent{
		Kind:       RelationshipChangeDelete,
		Value:      relation,
		FollowerID: relation.FollowerID,
		FolloweeID: relation.FolloweeID,
	}
}

func (c *RelationshipPairChanges) createBlock(relation *entity.BlockRelation) {
	if relation == nil {
		return
	}
	c.Block = BlockPersistenceIntent{
		Kind:      RelationshipChangeUpsert,
		Value:     relation,
		BlockerID: relation.BlockerID,
		BlockedID: relation.BlockedID,
	}
}

func (c *RelationshipPairChanges) deleteBlock(relation *entity.BlockRelation) {
	if relation == nil {
		return
	}
	c.Block = BlockPersistenceIntent{
		Kind:      RelationshipChangeDelete,
		Value:     relation,
		BlockerID: relation.BlockerID,
		BlockedID: relation.BlockedID,
	}
}

func (c *RelationshipPairChanges) addCounterDelta(userID string, delta entity.UserRelationshipCounterDelta) {
	if strings.TrimSpace(userID) == "" || delta.IsZero() {
		return
	}

	current := c.CounterDeltas[userID]
	current.FriendsCount += delta.FriendsCount
	current.FollowersCount += delta.FollowersCount
	current.FollowingCount += delta.FollowingCount
	current.BlockedCount += delta.BlockedCount
	current.PendingInCount += delta.PendingInCount
	current.PendingOutCount += delta.PendingOutCount
	c.CounterDeltas[userID] = current
}

func (c *RelationshipPairChanges) reset() {
	*c = newRelationshipPairChanges()
}

func CanonicalRelationshipPairAggregateID(actorID, targetID string) string {
	low, high := normalizePair(actorID, targetID)
	return low + ":" + high
}

func normalizePair(a, b string) (string, string) {
	if a <= b {
		return a, b
	}
	return b, a
}
