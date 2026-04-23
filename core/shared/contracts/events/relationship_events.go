package events

import "time"

const (
	EventRelationshipPairFriendRequestSent      = "EventRelationshipPairFriendRequestSent"
	EventRelationshipPairFriendRequestCancelled = "EventRelationshipPairFriendRequestCancelled"
	EventRelationshipPairFriendRequestAccepted  = "EventRelationshipPairFriendRequestAccepted"
	EventRelationshipPairFriendRequestRejected  = "EventRelationshipPairFriendRequestRejected"
	EventRelationshipPairFollowed               = "EventRelationshipPairFollowed"
	EventRelationshipPairUnfollowed             = "EventRelationshipPairUnfollowed"
	EventRelationshipPairUnfriended             = "EventRelationshipPairUnfriended"
	EventRelationshipPairBlocked                = "EventRelationshipPairBlocked"
	EventRelationshipPairUnblocked              = "EventRelationshipPairUnblocked"
)

type RelationshipPairFriendRequestSentEvent struct {
	RequestID   string
	RequesterID string
	AddresseeID string
	CreatedAt   time.Time
}

type RelationshipPairFriendRequestCancelledEvent struct {
	RequestID   string
	RequesterID string
	AddresseeID string
	CreatedAt   time.Time
	CancelledAt time.Time
}

type RelationshipPairFriendRequestAcceptedEvent struct {
	RequestID    string
	RequesterID  string
	AddresseeID  string
	CreatedAt    time.Time
	FriendshipID string
	AcceptedAt   time.Time
}

type RelationshipPairFriendRequestRejectedEvent struct {
	RequestID   string
	RequesterID string
	AddresseeID string
	CreatedAt   time.Time
	Reason      *string
	RejectedAt  time.Time
}

type RelationshipPairFollowedEvent struct {
	FollowID   string
	FollowerID string
	FolloweeID string
	CreatedAt  time.Time
}

type RelationshipPairUnfollowedEvent struct {
	FollowerID string
	FolloweeID string
}

type RelationshipPairUnfriendedEvent struct {
	UserID   string
	FriendID string
}

type RelationshipPairBlockedEvent struct {
	BlockID   string
	BlockerID string
	BlockedID string
	Reason    *string
	CreatedAt time.Time
}

type RelationshipPairUnblockedEvent struct {
	BlockerID string
	BlockedID string
}
