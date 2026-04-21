package cassandra

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	appCtx "wechat-clone/core/context"
	relationshipprojection "wechat-clone/core/modules/relationship/application/projection"
	"wechat-clone/core/modules/relationship/infra/projection/cassandra/views"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

type queryRepoImpl struct {
	session *gocql.Session
	tables  views.ProjectionTableNames
}

func NewProjectionRepo(appCtx *appCtx.AppContext) (relationshipprojection.ReadRepository, error) {
	tables := views.DefaultProjectionTableNames()
	if err := runProjectionMigrations(context.Background(), appCtx.GetCassandraSession(), tables); err != nil {
		return nil, stackErr.Error(err)
	}
	return &queryRepoImpl{session: appCtx.GetCassandraSession(), tables: tables}, nil
}

func (r *queryRepoImpl) GetPair(ctx context.Context, userA, userB string) (*relationshipprojection.RelationshipPairProjection, error) {
	pairID := canonicalPairID(userA, userB)
	if pairID == "" {
		return nil, nil
	}

	stmt := fmt.Sprintf(`SELECT pair_id, user_low_id, user_high_id,
		pending_request_id, pending_requester_id, pending_addressee_id, pending_request_created_at,
		friendship_id, friendship_created_at,
		low_follows_high, low_follows_high_at, high_follows_low, high_follows_low_at,
		low_blocks_high, low_blocks_high_at, high_blocks_low, high_blocks_low_at,
		created_at, updated_at
		FROM %s WHERE pair_id = ? LIMIT 1`, r.tables.PairByPair)

	var (
		projection              relationshipprojection.RelationshipPairProjection
		pendingRequestID        string
		pendingRequesterID      string
		pendingAddresseeID      string
		friendshipID            string
		pendingRequestCreatedAt *time.Time
		friendshipCreatedAt     *time.Time
		lowFollowsHighAt        *time.Time
		highFollowsLowAt        *time.Time
		lowBlocksHighAt         *time.Time
		highBlocksLowAt         *time.Time
	)

	if err := r.session.Query(stmt, pairID).WithContext(ctx).Consistency(gocql.One).Scan(
		&projection.PairID,
		&projection.UserLowID,
		&projection.UserHighID,
		&pendingRequestID,
		&pendingRequesterID,
		&pendingAddresseeID,
		&pendingRequestCreatedAt,
		&friendshipID,
		&friendshipCreatedAt,
		&projection.LowFollowsHigh,
		&lowFollowsHighAt,
		&projection.HighFollowsLow,
		&highFollowsLowAt,
		&projection.LowBlocksHigh,
		&lowBlocksHighAt,
		&projection.HighBlocksLow,
		&highBlocksLowAt,
		&projection.CreatedAt,
		&projection.UpdatedAt,
	); err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	projection.PendingRequestID = strings.TrimSpace(pendingRequestID)
	projection.PendingRequesterID = strings.TrimSpace(pendingRequesterID)
	projection.PendingAddresseeID = strings.TrimSpace(pendingAddresseeID)
	projection.PendingRequestCreatedAt = pendingRequestCreatedAt
	projection.FriendshipID = strings.TrimSpace(friendshipID)
	projection.FriendshipCreatedAt = friendshipCreatedAt
	projection.LowFollowsHighAt = lowFollowsHighAt
	projection.HighFollowsLowAt = highFollowsLowAt
	projection.LowBlocksHighAt = lowBlocksHighAt
	projection.HighBlocksLowAt = highBlocksLowAt
	return &projection, nil
}

func (r *queryRepoImpl) SavePair(ctx context.Context, projection *relationshipprojection.RelationshipPairProjection) error {
	if projection == nil {
		return stackErr.Error(fmt.Errorf("relationship pair projection is required"))
	}

	current, err := r.GetPair(ctx, projection.UserLowID, projection.UserHighID)
	if err != nil {
		return stackErr.Error(err)
	}

	if err := r.upsertPair(ctx, projection); err != nil {
		return stackErr.Error(err)
	}
	if err := r.syncFriendEdges(ctx, current, projection); err != nil {
		return stackErr.Error(err)
	}
	if err := r.syncFollowEdges(ctx, current, projection); err != nil {
		return stackErr.Error(err)
	}
	if err := r.syncBlockEdges(ctx, current, projection); err != nil {
		return stackErr.Error(err)
	}
	if err := r.syncPendingRequestEdges(ctx, current, projection); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *queryRepoImpl) ListFriends(ctx context.Context, userID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	return r.listByUserTable(ctx, r.tables.FriendsByUser, userID, cursor, limit)
}

func (r *queryRepoImpl) ListFollowers(ctx context.Context, userID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	return r.listByUserTable(ctx, r.tables.FollowersByUser, userID, cursor, limit)
}

func (r *queryRepoImpl) ListFollowing(ctx context.Context, userID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	return r.listByUserTable(ctx, r.tables.FollowingByUser, userID, cursor, limit)
}

func (r *queryRepoImpl) ListBlockedUsers(ctx context.Context, userID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	return r.listByUserTable(ctx, r.tables.BlocksByUser, userID, cursor, limit)
}

func (r *queryRepoImpl) ListIncomingFriendRequests(ctx context.Context, userID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	return r.listByUserTable(ctx, r.tables.IncomingRequestsByUser, userID, cursor, limit)
}

func (r *queryRepoImpl) ListOutgoingFriendRequests(ctx context.Context, userID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	return r.listByUserTable(ctx, r.tables.OutgoingRequestsByUser, userID, cursor, limit)
}

func (r *queryRepoImpl) ListMutualFriends(ctx context.Context, userID, targetUserID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	left, err := r.ListFriends(ctx, userID, "", 10000)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	right, err := r.ListFriends(ctx, targetUserID, "", 10000)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	rightSet := make(map[string]struct{}, len(right.Items))
	for _, item := range right.Items {
		rightSet[item] = struct{}{}
	}
	items := make([]string, 0, len(left.Items))
	for _, item := range left.Items {
		if _, ok := rightSet[item]; ok && item > cursor {
			items = append(items, item)
		}
	}
	sort.Strings(items)
	return paginateIDs(items, limit), nil
}

func (r *queryRepoImpl) CountFriends(ctx context.Context, userID string) (int64, error) {
	return r.countByUserTable(ctx, r.tables.FriendsByUser, userID)
}

func (r *queryRepoImpl) CountFollowers(ctx context.Context, userID string) (int64, error) {
	return r.countByUserTable(ctx, r.tables.FollowersByUser, userID)
}

func (r *queryRepoImpl) CountFollowing(ctx context.Context, userID string) (int64, error) {
	return r.countByUserTable(ctx, r.tables.FollowingByUser, userID)
}

func (r *queryRepoImpl) CountMutualFriends(ctx context.Context, userID, targetUserID string) (int64, error) {
	result, err := r.ListMutualFriends(ctx, userID, targetUserID, "", 10000)
	if err != nil {
		return 0, stackErr.Error(err)
	}
	return int64(len(result.Items)), nil
}

func (r *queryRepoImpl) upsertPair(ctx context.Context, projection *relationshipprojection.RelationshipPairProjection) error {
	stmt := fmt.Sprintf(`INSERT INTO %s (
		pair_id, user_low_id, user_high_id,
		pending_request_id, pending_requester_id, pending_addressee_id, pending_request_created_at,
		friendship_id, friendship_created_at,
		low_follows_high, low_follows_high_at, high_follows_low, high_follows_low_at,
		low_blocks_high, low_blocks_high_at, high_blocks_low, high_blocks_low_at,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, r.tables.PairByPair)

	return stackErr.Error(r.session.Query(
		stmt,
		projection.PairID,
		projection.UserLowID,
		projection.UserHighID,
		projection.PendingRequestID,
		projection.PendingRequesterID,
		projection.PendingAddresseeID,
		projection.PendingRequestCreatedAt,
		projection.FriendshipID,
		projection.FriendshipCreatedAt,
		projection.LowFollowsHigh,
		projection.LowFollowsHighAt,
		projection.HighFollowsLow,
		projection.HighFollowsLowAt,
		projection.LowBlocksHigh,
		projection.LowBlocksHighAt,
		projection.HighBlocksLow,
		projection.HighBlocksLowAt,
		projection.CreatedAt.UTC(),
		projection.UpdatedAt.UTC(),
	).WithContext(ctx).Exec())
}

func (r *queryRepoImpl) syncFriendEdges(ctx context.Context, current, next *relationshipprojection.RelationshipPairProjection) error {
	currentActive := current != nil && strings.TrimSpace(current.FriendshipID) != ""
	nextActive := next != nil && strings.TrimSpace(next.FriendshipID) != ""
	if currentActive == nextActive && (!nextActive || current.FriendshipID == next.FriendshipID) {
		return nil
	}
	if currentActive {
		if err := r.deletePairEdge(ctx, r.tables.FriendsByUser, current.UserLowID, current.UserHighID); err != nil {
			return stackErr.Error(err)
		}
		if err := r.deletePairEdge(ctx, r.tables.FriendsByUser, current.UserHighID, current.UserLowID); err != nil {
			return stackErr.Error(err)
		}
	}
	if nextActive {
		createdAt := derefTime(next.FriendshipCreatedAt, next.UpdatedAt)
		if err := r.upsertPairEdge(ctx, r.tables.FriendsByUser, next.UserLowID, next.UserHighID, createdAt); err != nil {
			return stackErr.Error(err)
		}
		if err := r.upsertPairEdge(ctx, r.tables.FriendsByUser, next.UserHighID, next.UserLowID, createdAt); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}

func (r *queryRepoImpl) syncFollowEdges(ctx context.Context, current, next *relationshipprojection.RelationshipPairProjection) error {
	if err := r.syncDirectionalEdge(ctx, r.tables.FollowingByUser, r.tables.FollowersByUser, current, next, next != nil && next.LowFollowsHigh, current != nil && current.LowFollowsHigh, next.UserLowID, next.UserHighID, derefTime(next.LowFollowsHighAt, next.UpdatedAt)); err != nil {
		return stackErr.Error(err)
	}
	if err := r.syncDirectionalEdge(ctx, r.tables.FollowingByUser, r.tables.FollowersByUser, current, next, next != nil && next.HighFollowsLow, current != nil && current.HighFollowsLow, next.UserHighID, next.UserLowID, derefTime(next.HighFollowsLowAt, next.UpdatedAt)); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *queryRepoImpl) syncBlockEdges(ctx context.Context, current, next *relationshipprojection.RelationshipPairProjection) error {
	if err := r.syncSingleDirectionalTable(ctx, r.tables.BlocksByUser, current != nil && current.LowBlocksHigh, next != nil && next.LowBlocksHigh, next.UserLowID, next.UserHighID, derefTime(next.LowBlocksHighAt, next.UpdatedAt)); err != nil {
		return stackErr.Error(err)
	}
	if err := r.syncSingleDirectionalTable(ctx, r.tables.BlocksByUser, current != nil && current.HighBlocksLow, next != nil && next.HighBlocksLow, next.UserHighID, next.UserLowID, derefTime(next.HighBlocksLowAt, next.UpdatedAt)); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *queryRepoImpl) syncPendingRequestEdges(ctx context.Context, current, next *relationshipprojection.RelationshipPairProjection) error {
	currentRequester := ""
	currentAddressee := ""
	currentRequestID := ""
	if current != nil {
		currentRequester = current.PendingRequesterID
		currentAddressee = current.PendingAddresseeID
		currentRequestID = current.PendingRequestID
	}
	nextRequester := ""
	nextAddressee := ""
	nextRequestID := ""
	if next != nil {
		nextRequester = next.PendingRequesterID
		nextAddressee = next.PendingAddresseeID
		nextRequestID = next.PendingRequestID
	}
	if currentRequestID != "" && (currentRequestID != nextRequestID || currentRequester != nextRequester || currentAddressee != nextAddressee) {
		if err := r.deletePairEdge(ctx, r.tables.OutgoingRequestsByUser, currentRequester, currentAddressee); err != nil {
			return stackErr.Error(err)
		}
		if err := r.deletePairEdge(ctx, r.tables.IncomingRequestsByUser, currentAddressee, currentRequester); err != nil {
			return stackErr.Error(err)
		}
	}
	if nextRequestID != "" {
		createdAt := derefTime(next.PendingRequestCreatedAt, next.UpdatedAt)
		if err := r.upsertRequestEdge(ctx, r.tables.OutgoingRequestsByUser, nextRequester, nextAddressee, nextRequestID, createdAt); err != nil {
			return stackErr.Error(err)
		}
		if err := r.upsertRequestEdge(ctx, r.tables.IncomingRequestsByUser, nextAddressee, nextRequester, nextRequestID, createdAt); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}

func (r *queryRepoImpl) syncDirectionalEdge(ctx context.Context, primaryTable, reverseTable string, current, next *relationshipprojection.RelationshipPairProjection, nextActive, currentActive bool, actorID, targetID string, createdAt time.Time) error {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(targetID) == "" {
		return nil
	}
	if currentActive == nextActive {
		return nil
	}
	if currentActive {
		if err := r.deletePairEdge(ctx, primaryTable, actorID, targetID); err != nil {
			return stackErr.Error(err)
		}
		if err := r.deletePairEdge(ctx, reverseTable, targetID, actorID); err != nil {
			return stackErr.Error(err)
		}
	}
	if nextActive {
		if err := r.upsertPairEdge(ctx, primaryTable, actorID, targetID, createdAt); err != nil {
			return stackErr.Error(err)
		}
		if err := r.upsertPairEdge(ctx, reverseTable, targetID, actorID, createdAt); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}

func (r *queryRepoImpl) syncSingleDirectionalTable(ctx context.Context, table string, currentActive, nextActive bool, actorID, targetID string, createdAt time.Time) error {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(targetID) == "" {
		return nil
	}
	if currentActive == nextActive {
		return nil
	}
	if currentActive {
		return stackErr.Error(r.deletePairEdge(ctx, table, actorID, targetID))
	}
	if nextActive {
		return stackErr.Error(r.upsertPairEdge(ctx, table, actorID, targetID, createdAt))
	}
	return nil
}

func (r *queryRepoImpl) upsertPairEdge(ctx context.Context, table, userID, counterpartID string, createdAt time.Time) error {
	stmt := fmt.Sprintf("INSERT INTO %s (user_id, counterpart_id, created_at) VALUES (?, ?, ?)", table)
	return stackErr.Error(r.session.Query(stmt, userID, counterpartID, createdAt.UTC()).WithContext(ctx).Exec())
}

func (r *queryRepoImpl) upsertRequestEdge(ctx context.Context, table, userID, counterpartID, requestID string, createdAt time.Time) error {
	stmt := fmt.Sprintf("INSERT INTO %s (user_id, counterpart_id, created_at, request_id) VALUES (?, ?, ?, ?)", table)
	return stackErr.Error(r.session.Query(stmt, userID, counterpartID, createdAt.UTC(), requestID).WithContext(ctx).Exec())
}

func (r *queryRepoImpl) deletePairEdge(ctx context.Context, table, userID, counterpartID string) error {
	stmt := fmt.Sprintf("DELETE FROM %s WHERE user_id = ? AND counterpart_id = ?", table)
	return stackErr.Error(r.session.Query(stmt, userID, counterpartID).WithContext(ctx).Exec())
}

func (r *queryRepoImpl) listByUserTable(ctx context.Context, table, userID, cursor string, limit int) (*relationshipprojection.RelationshipListResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	stmt := fmt.Sprintf("SELECT counterpart_id FROM %s WHERE user_id = ?", table)
	iter := r.session.Query(stmt, userID).WithContext(ctx).Iter()
	var counterpartID string
	items := make([]string, 0)
	for iter.Scan(&counterpartID) {
		id := strings.TrimSpace(counterpartID)
		if id != "" && id > cursor {
			items = append(items, id)
		}
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(err)
	}
	sort.Strings(items)
	return paginateIDs(items, limit), nil
}

func (r *queryRepoImpl) countByUserTable(ctx context.Context, table, userID string) (int64, error) {
	stmt := fmt.Sprintf("SELECT counterpart_id FROM %s WHERE user_id = ?", table)
	iter := r.session.Query(stmt, userID).WithContext(ctx).Iter()
	var (
		counterpartID string
		total         int64
	)
	for iter.Scan(&counterpartID) {
		total++
	}
	if err := iter.Close(); err != nil {
		return 0, stackErr.Error(err)
	}
	return total, nil
}

func paginateIDs(items []string, limit int) *relationshipprojection.RelationshipListResult {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if len(items) == 0 {
		return &relationshipprojection.RelationshipListResult{Items: []string{}}
	}
	result := &relationshipprojection.RelationshipListResult{}
	if len(items) > limit {
		result.Items = append(result.Items, items[:limit]...)
		result.NextCursor = items[limit]
	} else {
		result.Items = append(result.Items, items...)
	}
	result.Total = int64(len(items))
	return result
}

func canonicalPairID(userA, userB string) string {
	low, high := normalizePair(strings.TrimSpace(userA), strings.TrimSpace(userB))
	if low == "" && high == "" {
		return ""
	}
	return low + ":" + high
}

func normalizePair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func derefTime(value *time.Time, fallback time.Time) time.Time {
	if value != nil {
		return value.UTC()
	}
	return fallback.UTC()
}
