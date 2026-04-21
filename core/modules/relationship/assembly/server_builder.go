package assembly

import (
	"context"

	appCtx "wechat-clone/core/context"
	relationshipcommand "wechat-clone/core/modules/relationship/application/command"
	relationshipquery "wechat-clone/core/modules/relationship/application/query"
	relationshiprepo "wechat-clone/core/modules/relationship/infra/persistent/repository"
	relationshipReadRepos "wechat-clone/core/modules/relationship/infra/projection/cassandra"
	relationshipserver "wechat-clone/core/modules/relationship/transport/server"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/transport/http"
)

func buildHTTPServer(_ context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	relationshipRepos := relationshiprepo.NewRepoImpl(appContext)
	relationshipReadRepos, err := relationshipReadRepos.NewProjectionRepo(appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	sendFriendRequest := cqrs.NewDispatcher(relationshipcommand.NewSendFriendRequest(appContext, relationshipRepos))
	cancelFriendRequest := cqrs.NewDispatcher(relationshipcommand.NewCancelFriendRequest(appContext, relationshipRepos))
	acceptFriendRequest := cqrs.NewDispatcher(relationshipcommand.NewAcceptFriendRequest(appContext, relationshipRepos))
	rejectFriendRequest := cqrs.NewDispatcher(relationshipcommand.NewRejectFriendRequest(appContext, relationshipRepos))
	listIncomingFriendRequests := cqrs.NewDispatcher(relationshipquery.NewListIncomingFriendRequests(appContext, relationshipReadRepos))
	listOutgoingFriendRequests := cqrs.NewDispatcher(relationshipquery.NewListOutgoingFriendRequests(appContext, relationshipReadRepos))
	unfriendUser := cqrs.NewDispatcher(relationshipcommand.NewUnfriendUser(appContext, relationshipRepos))
	listFriends := cqrs.NewDispatcher(relationshipquery.NewListFriends(appContext, relationshipReadRepos))
	followUser := cqrs.NewDispatcher(relationshipcommand.NewFollowUser(appContext, relationshipRepos))
	unfollowUser := cqrs.NewDispatcher(relationshipcommand.NewUnfollowUser(appContext, relationshipRepos))
	listFollowers := cqrs.NewDispatcher(relationshipquery.NewListFollowers(appContext, relationshipReadRepos))
	listFollowing := cqrs.NewDispatcher(relationshipquery.NewListFollowing(appContext, relationshipReadRepos))
	blockUser := cqrs.NewDispatcher(relationshipcommand.NewBlockUser(appContext, relationshipRepos))
	unblockUser := cqrs.NewDispatcher(relationshipcommand.NewUnblockUser(appContext, relationshipRepos))
	listBlockedUsers := cqrs.NewDispatcher(relationshipquery.NewListBlockedUsers(appContext, relationshipReadRepos))
	getRelationshipStatus := cqrs.NewDispatcher(relationshipquery.NewGetRelationshipStatus(appContext, relationshipReadRepos))
	getMutualFriends := cqrs.NewDispatcher(relationshipquery.NewGetMutualFriends(appContext, relationshipReadRepos))
	getRelationshipSummary := cqrs.NewDispatcher(relationshipquery.NewGetRelationshipSummary(appContext, relationshipReadRepos))

	server, err := relationshipserver.NewHTTPServer(
		sendFriendRequest,
		cancelFriendRequest,
		acceptFriendRequest,
		rejectFriendRequest,
		listIncomingFriendRequests,
		listOutgoingFriendRequests,
		unfriendUser,
		listFriends,
		followUser,
		unfollowUser,
		listFollowers,
		listFollowing,
		blockUser,
		unblockUser,
		listBlockedUsers,
		getRelationshipStatus,
		getMutualFriends,
		getRelationshipSummary,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}
