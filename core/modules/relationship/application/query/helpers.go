package query

import (
	"context"
	"strings"

	"wechat-clone/core/modules/relationship/application/dto/out"
	relationshipprojection "wechat-clone/core/modules/relationship/application/projection"
	"wechat-clone/core/modules/relationship/support"
	"wechat-clone/core/shared/pkg/stackErr"
)

func currentAccountID(ctx context.Context) (string, error) {
	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return accountID, nil
}

func normalizeListTarget(currentUserID, requestedUserID string) string {
	requestedUserID = strings.TrimSpace(requestedUserID)
	if requestedUserID != "" {
		return requestedUserID
	}
	return strings.TrimSpace(currentUserID)
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 20
	}
	return limit
}

func buildRelationshipStatusResponse(
	currentUserID string,
	targetUserID string,
	pair *relationshipprojection.RelationshipPairProjection,
) *out.GetRelationshipStatusResponse {
	response := &out.GetRelationshipStatusResponse{
		IsSelf: strings.TrimSpace(currentUserID) == strings.TrimSpace(targetUserID),
	}
	if response.IsSelf {
		return response
	}
	if pair == nil {
		response.CanSendFriendRequest = true
		response.CanFollow = true
		return response
	}

	response.IsFriend = strings.TrimSpace(pair.FriendshipID) != ""
	response.IsFollowing = pairFollows(pair, currentUserID, targetUserID)
	response.IsFollower = pairFollows(pair, targetUserID, currentUserID)
	response.HasBlocked = pairBlocks(pair, currentUserID, targetUserID)
	response.IsBlockedBy = pairBlocks(pair, targetUserID, currentUserID)
	response.OutgoingFriendRequestPending = strings.TrimSpace(pair.PendingRequesterID) == strings.TrimSpace(currentUserID) &&
		strings.TrimSpace(pair.PendingAddresseeID) == strings.TrimSpace(targetUserID)
	response.IncomingFriendRequestPending = strings.TrimSpace(pair.PendingRequesterID) == strings.TrimSpace(targetUserID) &&
		strings.TrimSpace(pair.PendingAddresseeID) == strings.TrimSpace(currentUserID)
	response.CanSendFriendRequest = !response.IsFriend &&
		!response.HasBlocked &&
		!response.IsBlockedBy &&
		!response.OutgoingFriendRequestPending &&
		!response.IncomingFriendRequestPending
	response.CanFollow = !response.IsFollowing && !response.HasBlocked && !response.IsBlockedBy
	return response
}

func pairFollows(pair *relationshipprojection.RelationshipPairProjection, followerID, followeeID string) bool {
	if pair == nil {
		return false
	}
	low, high := normalizePairIDs(followerID, followeeID)
	if followerID == low && followeeID == high {
		return pair.LowFollowsHigh
	}
	return pair.HighFollowsLow
}

func pairBlocks(pair *relationshipprojection.RelationshipPairProjection, blockerID, blockedID string) bool {
	if pair == nil {
		return false
	}
	low, high := normalizePairIDs(blockerID, blockedID)
	if blockerID == low && blockedID == high {
		return pair.LowBlocksHigh
	}
	return pair.HighBlocksLow
}

func relationshipStatusLabel(status *out.GetRelationshipStatusResponse) string {
	if status == nil {
		return "none"
	}
	switch {
	case status.IsSelf:
		return "self"
	case status.HasBlocked:
		return "blocked"
	case status.IsBlockedBy:
		return "blocked_by"
	case status.IsFriend:
		return "friends"
	case status.OutgoingFriendRequestPending:
		return "outgoing_friend_request_pending"
	case status.IncomingFriendRequestPending:
		return "incoming_friend_request_pending"
	case status.IsFollowing && status.IsFollower:
		return "mutual_follow"
	case status.IsFollowing:
		return "following"
	case status.IsFollower:
		return "follower"
	default:
		return "none"
	}
}

func normalizePairIDs(userA, userB string) (string, string) {
	userA = strings.TrimSpace(userA)
	userB = strings.TrimSpace(userB)
	if userA < userB {
		return userA, userB
	}
	return userB, userA
}

func emptyListResult() *relationshipprojection.RelationshipListResult {
	return &relationshipprojection.RelationshipListResult{Items: []string{}}
}
