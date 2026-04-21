package views

type ProjectionTableNames struct {
	PairByPair           string
	FriendsByUser        string
	FollowersByUser      string
	FollowingByUser      string
	BlocksByUser         string
	IncomingRequestsByUser string
	OutgoingRequestsByUser string
	SchemaMigrations     string
}

func DefaultProjectionTableNames() ProjectionTableNames {
	return ProjectionTableNames{
		PairByPair:             "relationship_pair_projections_by_pair",
		FriendsByUser:          "relationship_friends_by_user",
		FollowersByUser:        "relationship_followers_by_user",
		FollowingByUser:        "relationship_following_by_user",
		BlocksByUser:           "relationship_blocks_by_user",
		IncomingRequestsByUser: "relationship_incoming_requests_by_user",
		OutgoingRequestsByUser: "relationship_outgoing_requests_by_user",
		SchemaMigrations:       "relationship_projection_schema_migrations",
	}
}
