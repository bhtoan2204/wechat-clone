package aggregate

import (
	"fmt"
	"time"

	"wechat-clone/core/modules/relationship/domain"
	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

type RelationshipPairAggregate struct {
	event.AggregateRoot

	state   relationshipPairState
	changes RelationshipPairChanges
}

func NewRelationshipPair(snapshot RelationshipPairSnapshot) (*RelationshipPairAggregate, error) {
	if snapshot.ActorID == "" || snapshot.TargetID == "" {
		return nil, stackErr.Error(domain.ErrEmpty)
	}

	agg := &RelationshipPairAggregate{
		state:   newRelationshipPairState(snapshot),
		changes: newRelationshipPairChanges(),
	}

	aggregateID := CanonicalRelationshipPairAggregateID(snapshot.ActorID, snapshot.TargetID)
	if err := event.InitAggregate(&agg.AggregateRoot, agg, aggregateID); err != nil {
		return nil, stackErr.Error(err)
	}
	agg.Root().SetInternal(aggregateID, snapshot.AggregateVersion, snapshot.AggregateVersion)
	return agg, nil
}

func (a *RelationshipPairAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventRelationshipPairFriendRequestSent{},
		&EventRelationshipPairFriendRequestCancelled{},
		&EventRelationshipPairFriendRequestAccepted{},
		&EventRelationshipPairFriendRequestRejected{},
		&EventRelationshipPairFollowed{},
		&EventRelationshipPairUnfollowed{},
		&EventRelationshipPairUnfriended{},
		&EventRelationshipPairBlocked{},
		&EventRelationshipPairUnblocked{},
	)
}

func (a *RelationshipPairAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *EventRelationshipPairFriendRequestSent:
		return stackErr.Error(a.applyFriendRequestSent(data))
	case *EventRelationshipPairFriendRequestCancelled:
		return stackErr.Error(a.applyFriendRequestCancelled(data))
	case *EventRelationshipPairFriendRequestAccepted:
		return stackErr.Error(a.applyFriendRequestAccepted(data))
	case *EventRelationshipPairFriendRequestRejected:
		return stackErr.Error(a.applyFriendRequestRejected(data))
	case *EventRelationshipPairFollowed:
		return stackErr.Error(a.applyFollowed(data))
	case *EventRelationshipPairUnfollowed:
		return stackErr.Error(a.applyUnfollowed())
	case *EventRelationshipPairUnfriended:
		return stackErr.Error(a.applyUnfriended())
	case *EventRelationshipPairBlocked:
		return stackErr.Error(a.applyBlocked(data))
	case *EventRelationshipPairUnblocked:
		return stackErr.Error(a.applyUnblocked())
	default:
		return event.ErrUnsupportedEventType
	}
}

func (a *RelationshipPairAggregate) SendFriendRequest(requestID string, now time.Time) error {
	if err := a.state.toPolicyState().EnsureCanSendFriendRequest(); err != nil {
		return stackErr.Error(err)
	}

	friendRequest, err := NewFriendRequest(requestID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := friendRequest.Create(a.state.actorID, a.state.targetID, nil, now); err != nil {
		return stackErr.Error(err)
	}
	a.state.friendRequest = friendRequest

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairFriendRequestSent{
		RequestID:   requestID,
		RequesterID: a.state.actorID,
		AddresseeID: a.state.targetID,
		CreatedAt:   now,
	}))
}

func (a *RelationshipPairAggregate) CancelFriendRequest(now time.Time) error {
	if err := a.state.toPolicyState().EnsureCanCancelFriendRequest(); err != nil {
		return stackErr.Error(err)
	}
	if a.state.friendRequest == nil {
		return stackErr.Error(fmt.Errorf("friend request aggregate is required"))
	}
	if err := a.state.friendRequest.CancelRequest(now); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairFriendRequestCancelled{
		RequestID:   a.state.friendRequest.AggregateID(),
		RequesterID: a.state.friendRequest.RequesterID,
		AddresseeID: a.state.friendRequest.AddresseeID,
		CreatedAt:   a.state.friendRequest.CreatedAt,
		CancelledAt: now,
	}))
}

func (a *RelationshipPairAggregate) AcceptFriendRequest(friendshipID string, now time.Time) error {
	if err := a.state.toPolicyState().EnsureCanAcceptFriendRequest(); err != nil {
		return stackErr.Error(err)
	}
	if a.state.friendRequest == nil {
		return stackErr.Error(fmt.Errorf("friend request aggregate is required"))
	}
	if err := a.state.friendRequest.AcceptRequest(now); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairFriendRequestAccepted{
		RequestID:    a.state.friendRequest.AggregateID(),
		RequesterID:  a.state.friendRequest.RequesterID,
		AddresseeID:  a.state.friendRequest.AddresseeID,
		CreatedAt:    a.state.friendRequest.CreatedAt,
		FriendshipID: friendshipID,
		AcceptedAt:   now,
	}))
}

func (a *RelationshipPairAggregate) RejectFriendRequest(reason *string, now time.Time) error {
	if err := a.state.toPolicyState().EnsureCanRejectFriendRequest(); err != nil {
		return stackErr.Error(err)
	}
	if a.state.friendRequest == nil {
		return stackErr.Error(fmt.Errorf("friend request aggregate is required"))
	}
	if err := a.state.friendRequest.RejectRequest(reason, now); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairFriendRequestRejected{
		RequestID:   a.state.friendRequest.AggregateID(),
		RequesterID: a.state.friendRequest.RequesterID,
		AddresseeID: a.state.friendRequest.AddresseeID,
		CreatedAt:   a.state.friendRequest.CreatedAt,
		Reason:      reason,
		RejectedAt:  now,
	}))
}

func (a *RelationshipPairAggregate) Follow(followID string, now time.Time) error {
	if err := a.state.toPolicyState().EnsureCanFollow(); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairFollowed{
		FollowID:   followID,
		FollowerID: a.state.actorID,
		FolloweeID: a.state.targetID,
		CreatedAt:  now,
	}))
}

func (a *RelationshipPairAggregate) Unfollow() error {
	if err := a.state.toPolicyState().EnsureCanUnfollow(); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairUnfollowed{
		FollowerID: a.state.actorID,
		FolloweeID: a.state.targetID,
	}))
}

func (a *RelationshipPairAggregate) Unfriend() error {
	if err := a.state.toPolicyState().EnsureCanUnfriend(); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairUnfriended{
		UserID:   a.state.actorID,
		FriendID: a.state.targetID,
	}))
}

func (a *RelationshipPairAggregate) Block(blockID string, reason *string, now time.Time) error {
	if err := a.state.toPolicyState().EnsureCanBlock(); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairBlocked{
		BlockID:   blockID,
		BlockerID: a.state.actorID,
		BlockedID: a.state.targetID,
		Reason:    reason,
		CreatedAt: now,
	}))
}

func (a *RelationshipPairAggregate) Unblock() error {
	if err := a.state.toPolicyState().EnsureCanUnblock(); err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(a.ApplyChange(a, &EventRelationshipPairUnblocked{
		BlockerID: a.state.actorID,
		BlockedID: a.state.targetID,
	}))
}

func (a *RelationshipPairAggregate) ActorID() string {
	return a.state.actorID
}

func (a *RelationshipPairAggregate) TargetID() string {
	return a.state.targetID
}

func (a *RelationshipPairAggregate) Changes() RelationshipPairChanges {
	return a.changes
}

func (a *RelationshipPairAggregate) FriendRequest() *FriendRequestAggregate {
	return a.state.friendRequest
}

func (a *RelationshipPairAggregate) FriendshipCreated() *entity.Friendship {
	if a.changes.Friendship.Kind != RelationshipChangeUpsert {
		return nil
	}
	return a.changes.Friendship.Value
}

func (a *RelationshipPairAggregate) FollowCreated() *entity.FollowRelation {
	if a.changes.Following.Kind != RelationshipChangeUpsert {
		return nil
	}
	return a.changes.Following.Value
}

func (a *RelationshipPairAggregate) BlockCreated() *entity.BlockRelation {
	if a.changes.Block.Kind != RelationshipChangeUpsert {
		return nil
	}
	return a.changes.Block.Value
}

func (a *RelationshipPairAggregate) MarkPersisted() {
	a.AggregateRoot.MarkPersisted()
	a.changes.reset()
}

func (a *RelationshipPairAggregate) applyFriendRequestSent(data *EventRelationshipPairFriendRequestSent) error {
	if a.state.friendRequest == nil || a.state.friendRequest.AggregateID() != data.RequestID {
		friendRequest, err := buildFriendRequestAggregate(
			data.RequestID,
			data.RequesterID,
			data.AddresseeID,
			entity.FriendRequestStatusPending,
			data.CreatedAt,
			nil,
			nil,
			nil,
		)
		if err != nil {
			return stackErr.Error(err)
		}
		a.state.friendRequest = friendRequest
	}

	a.changes.trackFriendRequest(a.state.friendRequest)
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{PendingOutCount: 1})
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{PendingInCount: 1})
	return nil
}

func (a *RelationshipPairAggregate) applyFriendRequestCancelled(data *EventRelationshipPairFriendRequestCancelled) error {
	friendRequest, err := a.ensureFriendRequestAggregate(
		data.RequestID,
		data.RequesterID,
		data.AddresseeID,
		entity.FriendRequestStatusCancelled,
		data.CreatedAt,
		nil,
		&data.CancelledAt,
		nil,
	)
	if err != nil {
		return stackErr.Error(err)
	}

	a.state.friendRequest = friendRequest
	a.changes.trackFriendRequest(friendRequest)
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{PendingOutCount: -1})
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{PendingInCount: -1})
	return nil
}

func (a *RelationshipPairAggregate) applyFriendRequestAccepted(data *EventRelationshipPairFriendRequestAccepted) error {
	friendRequest, err := a.ensureFriendRequestAggregate(
		data.RequestID,
		data.RequesterID,
		data.AddresseeID,
		entity.FriendRequestStatusAccepted,
		data.CreatedAt,
		&data.AcceptedAt,
		nil,
		nil,
	)
	if err != nil {
		return stackErr.Error(err)
	}

	friendship, err := entity.NewFriendship(data.FriendshipID, a.state.actorID, a.state.targetID, &data.RequestID, data.AcceptedAt)
	if err != nil {
		return stackErr.Error(err)
	}

	a.state.friendRequest = friendRequest
	a.state.friendship = friendship
	a.changes.trackFriendRequest(friendRequest)
	a.changes.createFriendship(friendship)
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{
		FriendsCount:    1,
		PendingOutCount: -1,
	})
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{
		FriendsCount:   1,
		PendingInCount: -1,
	})
	return nil
}

func (a *RelationshipPairAggregate) applyFriendRequestRejected(data *EventRelationshipPairFriendRequestRejected) error {
	friendRequest, err := a.ensureFriendRequestAggregate(
		data.RequestID,
		data.RequesterID,
		data.AddresseeID,
		entity.FriendRequestStatusRejected,
		data.CreatedAt,
		&data.RejectedAt,
		nil,
		data.Reason,
	)
	if err != nil {
		return stackErr.Error(err)
	}

	a.state.friendRequest = friendRequest
	a.changes.trackFriendRequest(friendRequest)
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{PendingOutCount: -1})
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{PendingInCount: -1})
	return nil
}

func (a *RelationshipPairAggregate) applyFollowed(data *EventRelationshipPairFollowed) error {
	relation, err := entity.NewFollowRelation(data.FollowID, data.FollowerID, data.FolloweeID, data.CreatedAt)
	if err != nil {
		return stackErr.Error(err)
	}

	a.state.following = relation
	a.changes.createFollowing(relation)
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{FollowingCount: 1})
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{FollowersCount: 1})
	return nil
}

func (a *RelationshipPairAggregate) applyUnfollowed() error {
	a.changes.deleteFollowing(a.state.following)
	a.state.following = nil
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{FollowingCount: -1})
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{FollowersCount: -1})
	return nil
}

func (a *RelationshipPairAggregate) applyUnfriended() error {
	a.changes.deleteFriendship(a.state.friendship)
	a.state.friendship = nil
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{FriendsCount: -1})
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{FriendsCount: -1})
	return nil
}

func (a *RelationshipPairAggregate) applyBlocked(data *EventRelationshipPairBlocked) error {
	if err := a.applyBlockRelation(data); err != nil {
		return stackErr.Error(err)
	}
	a.resolvePendingRequestForBlock(data.CreatedAt)
	a.removeFriendshipForBlock()
	a.removeOutgoingFollowForBlock()
	a.removeIncomingFollowForBlock()
	return nil
}

func (a *RelationshipPairAggregate) applyBlockRelation(data *EventRelationshipPairBlocked) error {
	relation, err := entity.NewBlockRelation(data.BlockID, data.BlockerID, data.BlockedID, data.Reason, data.CreatedAt)
	if err != nil {
		return stackErr.Error(err)
	}

	a.state.blocking = relation
	a.changes.createBlock(relation)
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{BlockedCount: 1})
	return nil
}

func (a *RelationshipPairAggregate) resolvePendingRequestForBlock(now time.Time) {
	if a.state.hasPendingOutgoingRequest() {
		if err := a.state.friendRequest.CancelRequest(now); err == nil {
			a.changes.trackFriendRequest(a.state.friendRequest)
			a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{PendingOutCount: -1})
			a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{PendingInCount: -1})
		}
	}

	if a.state.hasPendingIncomingRequest() {
		reason := "blocked by user"
		if err := a.state.friendRequest.RejectRequest(&reason, now); err == nil {
			a.changes.trackFriendRequest(a.state.friendRequest)
			a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{PendingOutCount: -1})
			a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{PendingInCount: -1})
		}
	}
}

func (a *RelationshipPairAggregate) removeFriendshipForBlock() {
	if a.state.friendship == nil {
		return
	}

	a.changes.deleteFriendship(a.state.friendship)
	a.state.friendship = nil
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{FriendsCount: -1})
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{FriendsCount: -1})
}

func (a *RelationshipPairAggregate) removeOutgoingFollowForBlock() {
	if a.state.following == nil {
		return
	}

	a.changes.deleteFollowing(a.state.following)
	a.state.following = nil
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{FollowingCount: -1})
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{FollowersCount: -1})
}

func (a *RelationshipPairAggregate) removeIncomingFollowForBlock() {
	if a.state.followedBy == nil {
		return
	}

	a.changes.deleteFollowedBy(a.state.followedBy)
	a.state.followedBy = nil
	a.changes.addCounterDelta(a.state.targetID, entity.UserRelationshipCounterDelta{FollowingCount: -1})
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{FollowersCount: -1})
}

func (a *RelationshipPairAggregate) applyUnblocked() error {
	a.changes.deleteBlock(a.state.blocking)
	a.state.blocking = nil
	a.changes.addCounterDelta(a.state.actorID, entity.UserRelationshipCounterDelta{BlockedCount: -1})
	return nil
}

func (a *RelationshipPairAggregate) ensureFriendRequestAggregate(
	requestID string,
	requesterID string,
	addresseeID string,
	status entity.FriendRequestStatus,
	createdAt time.Time,
	respondedAt *time.Time,
	cancelledAt *time.Time,
	rejectedReason *string,
) (*FriendRequestAggregate, error) {
	if a.state.friendRequest != nil {
		return a.state.friendRequest, nil
	}

	friendRequest, err := buildFriendRequestAggregate(
		requestID,
		requesterID,
		addresseeID,
		status,
		createdAt,
		respondedAt,
		cancelledAt,
		rejectedReason,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return friendRequest, nil
}

func buildFriendRequestAggregate(
	requestID string,
	requesterID string,
	addresseeID string,
	status entity.FriendRequestStatus,
	createdAt time.Time,
	respondedAt *time.Time,
	cancelledAt *time.Time,
	rejectedReason *string,
) (*FriendRequestAggregate, error) {
	friendRequest, err := NewFriendRequest(requestID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := friendRequest.SetFriendRequest(&entity.FriendRequest{
		ID:             requestID,
		RequesterID:    requesterID,
		AddresseeID:    addresseeID,
		Status:         status,
		CreatedAt:      createdAt,
		RespondedAt:    respondedAt,
		CancelledAt:    cancelledAt,
		RejectedReason: rejectedReason,
	}); err != nil {
		return nil, stackErr.Error(err)
	}
	return friendRequest, nil
}
