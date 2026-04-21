package projection

import "time"

type RelationshipPairProjection struct {
	PairID                  string
	UserLowID               string
	UserHighID              string
	PendingRequestID        string
	PendingRequesterID      string
	PendingAddresseeID      string
	PendingRequestCreatedAt *time.Time
	FriendshipID            string
	FriendshipCreatedAt     *time.Time
	LowFollowsHigh          bool
	LowFollowsHighAt        *time.Time
	HighFollowsLow          bool
	HighFollowsLowAt        *time.Time
	LowBlocksHigh           bool
	LowBlocksHighAt         *time.Time
	HighBlocksLow           bool
	HighBlocksLowAt         *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type RelationshipListResult struct {
	Items      []string
	NextCursor string
	Total      int64
}
