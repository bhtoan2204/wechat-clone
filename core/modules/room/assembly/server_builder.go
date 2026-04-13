package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	roomcommand "go-socket/core/modules/room/application/command"
	roomquery "go-socket/core/modules/room/application/query"
	roomservice "go-socket/core/modules/room/application/service"
	roomrepo "go-socket/core/modules/room/infra/persistent/repository"
	roomprojection "go-socket/core/modules/room/infra/projection/cassandra"
	roomserver "go-socket/core/modules/room/transport/server"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
	modruntime "go-socket/core/shared/runtime"
	"go-socket/core/shared/transport/http"
)

func buildHTTPServer(ctx context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	roomRepos, err := roomrepo.NewRepoImpl(appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	roomReadRepos, err := roomprojection.NewQueryRepoImpl(
		appContext.GetConfig().CassandraConfig,
		appContext.GetCassandraSession(),
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	roomQueryService := roomservice.NewRoomQueryService(roomReadRepos)
	chatQueryService := roomservice.NewChatQueryService(roomReadRepos, appContext.GetRedisClient())
	createRoom := cqrs.NewDispatcher(roomcommand.NewCreateRoomHandler(roomRepos))
	updateRoom := cqrs.NewDispatcher(roomcommand.NewUpdateRoomHandler(roomRepos))
	deleteRoom := cqrs.NewDispatcher(roomcommand.NewDeleteRoomHandler(roomRepos))
	getRoom := cqrs.NewDispatcher(roomquery.NewGetRoomHandler(roomQueryService))
	listRoom := cqrs.NewDispatcher(roomquery.NewListRoomHandler(roomQueryService))
	createDirectConversation := cqrs.NewDispatcher(roomcommand.NewCreateDirectConversationHandler(roomRepos))
	createGroupChat := cqrs.NewDispatcher(roomcommand.NewCreateGroupChatHandler(roomRepos))
	updateGroupChat := cqrs.NewDispatcher(roomcommand.NewUpdateGroupChatHandler(roomRepos))
	addChatMember := cqrs.NewDispatcher(roomcommand.NewAddChatMemberHandler(roomRepos))
	removeChatMember := cqrs.NewDispatcher(roomcommand.NewRemoveChatMemberHandler(roomRepos))
	pinChatMessage := cqrs.NewDispatcher(roomcommand.NewPinChatMessageHandler(roomRepos))
	sendChatMessage := cqrs.NewDispatcher(roomcommand.NewSendChatMessageHandler(roomRepos))
	editChatMessage := cqrs.NewDispatcher(roomcommand.NewEditChatMessageHandler(roomRepos))
	deleteChatMessage := cqrs.NewDispatcher(roomcommand.NewDeleteChatMessageHandler(roomRepos))
	forwardChatMessage := cqrs.NewDispatcher(roomcommand.NewForwardChatMessageHandler(roomRepos))
	markChatMessageStatus := cqrs.NewDispatcher(roomcommand.NewMarkChatMessageStatusHandler(roomRepos))
	listChatConversations := cqrs.NewDispatcher(roomquery.NewListChatConversationsHandler(chatQueryService))
	getChatConversation := cqrs.NewDispatcher(roomquery.NewGetChatConversationHandler(chatQueryService))
	listChatMessages := cqrs.NewDispatcher(roomquery.NewListChatMessagesHandler(chatQueryService))
	searchChatMentions := cqrs.NewDispatcher(roomquery.NewSearchChatMentionsHandler(chatQueryService))
	getChatPresence := cqrs.NewDispatcher(roomquery.NewGetChatPresenceHandler(chatQueryService))
	server, err := roomserver.NewHTTPServer(
		createRoom,
		listRoom,
		getRoom,
		updateRoom,
		deleteRoom,
		createDirectConversation,
		createGroupChat,
		updateGroupChat,
		listChatConversations,
		getChatConversation,
		listChatMessages,
		searchChatMentions,
		sendChatMessage,
		editChatMessage,
		deleteChatMessage,
		forwardChatMessage,
		markChatMessageStatus,
		addChatMember,
		removeChatMember,
		pinChatMessage,
		getChatPresence,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return server, nil
}

func buildProjectionRuntime(cfg *config.Config, appContext *appCtx.AppContext) (modruntime.Module, error) {
	accountProjection, err := buildProjectionHandler(cfg, appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	servingProjection, err := buildServingProjectionProcessor(cfg, appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return modruntime.NewComposite(accountProjection, servingProjection), nil
}
