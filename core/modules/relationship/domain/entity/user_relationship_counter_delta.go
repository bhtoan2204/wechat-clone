package entity

type UserRelationshipCounterDelta struct {
	FriendsCount    int64
	FollowersCount  int64
	FollowingCount  int64
	BlockedCount    int64
	PendingInCount  int64
	PendingOutCount int64
}

func (d UserRelationshipCounterDelta) IsZero() bool {
	return d.FriendsCount == 0 &&
		d.FollowersCount == 0 &&
		d.FollowingCount == 0 &&
		d.BlockedCount == 0 &&
		d.PendingInCount == 0 &&
		d.PendingOutCount == 0
}
