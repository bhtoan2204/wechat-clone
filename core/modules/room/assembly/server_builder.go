package assembly

import (
	"context"
	appCtx "go-socket/core/context"
	roomcommand "go-socket/core/modules/room/application/command"
	roomquery "go-socket/core/modules/room/application/query"
	roomservice "go-socket/core/modules/room/application/service"
	roomrepo "go-socket/core/modules/room/infra/persistent/repository"
	roomserver "go-socket/core/modules/room/transport/server"
	roomsocket "go-socket/core/modules/room/transport/websocket"
	"go-socket/core/shared/pkg/cqrs"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http"
)

func BuildHTTPServer(ctx context.Context, appContext *appCtx.AppContext) (http.HTTPServer, error) {
	roomRepos := roomrepo.NewRepoImpl(appContext)
	roomAggregateService := roomservice.NewRoomAggregateService()
	roomCommandService := roomservice.NewRoomCommandService(roomRepos, roomAggregateService)
	messageCommandService := roomservice.NewMessageCommandService(roomRepos, roomAggregateService)
	roomQueryService := roomservice.NewRoomQueryService(roomRepos)
	chatQueryService := roomservice.NewChatQueryService(roomRepos, appContext.GetRedisClient())
	createRoom := cqrs.NewDispatcher(roomcommand.NewCreateRoomHandler(roomCommandService))
	updateRoom := cqrs.NewDispatcher(roomcommand.NewUpdateRoomHandler(roomCommandService))
	deleteRoom := cqrs.NewDispatcher(roomcommand.NewDeleteRoomHandler(roomCommandService))
	getRoom := cqrs.NewDispatcher(roomquery.NewGetRoomHandler(roomQueryService))
	listRoom := cqrs.NewDispatcher(roomquery.NewListRoomHandler(roomQueryService))
	roomHub := roomsocket.NewHub(ctx, appContext)
	createDirectConversation := cqrs.NewDispatcher(roomcommand.NewCreateDirectConversationHandler(roomCommandService))
	createGroupChat := cqrs.NewDispatcher(roomcommand.NewCreateGroupChatHandler(roomCommandService))
	updateGroupChat := cqrs.NewDispatcher(roomcommand.NewUpdateGroupChatHandler(roomCommandService))
	addChatMember := cqrs.NewDispatcher(roomcommand.NewAddChatMemberHandler(roomCommandService))
	removeChatMember := cqrs.NewDispatcher(roomcommand.NewRemoveChatMemberHandler(roomCommandService))
	pinChatMessage := cqrs.NewDispatcher(roomcommand.NewPinChatMessageHandler(roomCommandService))
	sendChatMessage := cqrs.NewDispatcher(roomcommand.NewSendChatMessageHandler(messageCommandService))
	editChatMessage := cqrs.NewDispatcher(roomcommand.NewEditChatMessageHandler(messageCommandService))
	deleteChatMessage := cqrs.NewDispatcher(roomcommand.NewDeleteChatMessageHandler(messageCommandService))
	forwardChatMessage := cqrs.NewDispatcher(roomcommand.NewForwardChatMessageHandler(messageCommandService))
	markChatMessageStatus := cqrs.NewDispatcher(roomcommand.NewMarkChatMessageStatusHandler(messageCommandService))
	listChatConversations := cqrs.NewDispatcher(roomquery.NewListChatConversationsHandler(chatQueryService))
	getChatConversation := cqrs.NewDispatcher(roomquery.NewGetChatConversationHandler(chatQueryService))
	listChatMessages := cqrs.NewDispatcher(roomquery.NewListChatMessagesHandler(chatQueryService))
	getChatPresence := cqrs.NewDispatcher(roomquery.NewGetChatPresenceHandler(chatQueryService))
	server, err := roomserver.NewHTTPServer(
		createRoom,
		updateRoom,
		deleteRoom,
		getRoom,
		listRoom,
		createDirectConversation,
		createGroupChat,
		updateGroupChat,
		addChatMember,
		removeChatMember,
		pinChatMessage,
		listChatConversations,
		getChatConversation,
		listChatMessages,
		sendChatMessage,
		editChatMessage,
		deleteChatMessage,
		forwardChatMessage,
		markChatMessageStatus,
		getChatPresence,
		roomHub,
	)
	if err != nil {
		return nil, stackerr.Error(err)
	}

	return server, nil
}
