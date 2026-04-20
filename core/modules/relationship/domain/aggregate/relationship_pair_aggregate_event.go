package aggregate

import "time"

type EventRelationshipPairFriendRequestSent struct {
	RequestID   string
	RequesterID string
	AddresseeID string
	CreatedAt   time.Time
}

type EventRelationshipPairFriendRequestCancelled struct {
	RequestID   string
	RequesterID string
	AddresseeID string
	CreatedAt   time.Time
	CancelledAt time.Time
}

type EventRelationshipPairFriendRequestAccepted struct {
	RequestID    string
	RequesterID  string
	AddresseeID  string
	CreatedAt    time.Time
	FriendshipID string
	AcceptedAt   time.Time
}

type EventRelationshipPairFriendRequestRejected struct {
	RequestID   string
	RequesterID string
	AddresseeID string
	CreatedAt   time.Time
	Reason      *string
	RejectedAt  time.Time
}

type EventRelationshipPairFollowed struct {
	FollowID   string
	FollowerID string
	FolloweeID string
	CreatedAt  time.Time
}

type EventRelationshipPairUnfollowed struct {
	FollowerID string
	FolloweeID string
}

type EventRelationshipPairUnfriended struct {
	UserID   string
	FriendID string
}

type EventRelationshipPairBlocked struct {
	BlockID   string
	BlockerID string
	BlockedID string
	Reason    *string
	CreatedAt time.Time
}

type EventRelationshipPairUnblocked struct {
	BlockerID string
	BlockedID string
}
