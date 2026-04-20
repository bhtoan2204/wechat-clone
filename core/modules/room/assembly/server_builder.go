package assembly

import (
	"context"
	appCtx "wechat-clone/core/context"
	roomcommand "wechat-clone/core/modules/room/application/command"
	roomquery "wechat-clone/core/modules/room/application/query"
	roomservice "wechat-clone/core/modules/room/application/service"
	roomrepo "wechat-clone/core/modules/room/infra/persistent/repository"
	roomprojection "wechat-clone/core/modules/room/infra/projection/cassandra"
	roomserver "wechat-clone/core/modules/room/transport/server"
	roomsocket "wechat-clone/core/modules/room/transport/websocket"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
	modruntime "wechat-clone/core/shared/runtime"
	"wechat-clone/core/shared/transport/http"
	sharedsocket "wechat-clone/core/shared/transport/websocket"
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
	roomService := roomservice.NewService(appContext, roomReadRepos)
	createDirectConversation := cqrs.NewDispatcher(roomcommand.NewCreateDirectConversationHandler(roomRepos))
	createGroupChat := cqrs.NewDispatcher(roomcommand.NewCreateGroupChatHandler(roomRepos))
	updateGroupChat := cqrs.NewDispatcher(roomcommand.NewUpdateGroupChatHandler(roomRepos, roomService))
	addChatMember := cqrs.NewDispatcher(roomcommand.NewAddChatMemberHandler(roomRepos, roomService))
	removeChatMember := cqrs.NewDispatcher(roomcommand.NewRemoveChatMemberHandler(roomRepos, roomService))
	pinChatMessage := cqrs.NewDispatcher(roomcommand.NewPinChatMessageHandler(roomRepos, roomService))
	sendChatMessage := cqrs.NewDispatcher(roomcommand.NewSendChatMessageHandler(roomRepos, roomService))
	editChatMessage := cqrs.NewDispatcher(roomcommand.NewEditChatMessageHandler(roomRepos, roomService))
	deleteChatMessage := cqrs.NewDispatcher(roomcommand.NewDeleteChatMessageHandler(roomRepos, roomService))
	forwardChatMessage := cqrs.NewDispatcher(roomcommand.NewForwardChatMessageHandler(roomRepos, roomService))
	markChatMessageStatus := cqrs.NewDispatcher(roomcommand.NewMarkChatMessageStatusHandler(roomRepos, roomService))
	listChatConversations := cqrs.NewDispatcher(roomquery.NewListChatConversationsHandler(roomService))
	getChatConversation := cqrs.NewDispatcher(roomquery.NewGetChatConversationHandler(roomService))
	listChatMessages := cqrs.NewDispatcher(roomquery.NewListChatMessagesHandler(roomService))
	searchChatMentions := cqrs.NewDispatcher(roomquery.NewSearchChatMentionsHandler(roomService))
	getChatPresence := cqrs.NewDispatcher(roomquery.NewGetChatPresenceHandler(roomService))
	createChatMessagePresignedURL := cqrs.NewDispatcher(roomcommand.NewCreateChatMessagePresignedURLHandler(appContext, roomRepos))
	getChatMessageMedia := cqrs.NewDispatcher(roomquery.NewGetChatMessageMediaHandler(appContext, roomRepos))
	toggleChatMessageReaction := cqrs.NewDispatcher(roomcommand.NewToggleChatMessageReactionHandler(roomRepos, roomService))
	socketHub := roomsocket.NewHub(ctx, appContext)
	socketUpgrader := sharedsocket.NewUpgrader()
	socketHandler := roomsocket.NewWSHandler(appContext, socketHub, socketUpgrader)
	server, err := roomserver.NewHTTPServer(
		createDirectConversation,
		createGroupChat,
		updateGroupChat,
		listChatConversations,
		getChatConversation,
		listChatMessages,
		searchChatMentions,
		createChatMessagePresignedURL,
		getChatMessageMedia,
		sendChatMessage,
		toggleChatMessageReaction,
		editChatMessage,
		deleteChatMessage,
		forwardChatMessage,
		markChatMessageStatus,
		addChatMember,
		removeChatMember,
		pinChatMessage,
		getChatPresence,
		socketHandler.Handle,
		socketHub.Close,
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
