package server

import (
	"context"
	"fmt"
	roomin "go-socket/core/modules/room/application/dto/in"
	roomout "go-socket/core/modules/room/application/dto/out"
	roomhttp "go-socket/core/modules/room/transport/http"
	roomsocket "go-socket/core/modules/room/transport/websocket"
	"go-socket/core/shared/pkg/cqrs"
	stackerr "go-socket/core/shared/pkg/stackErr"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type roomServer struct {
	createRoom               cqrs.Dispatcher[*roomin.CreateRoomRequest, *roomout.CreateRoomResponse]
	updateRoom               cqrs.Dispatcher[*roomin.UpdateRoomRequest, *roomout.UpdateRoomResponse]
	deleteRoom               cqrs.Dispatcher[*roomin.DeleteRoomRequest, *roomout.DeleteRoomResponse]
	getRoom                  cqrs.Dispatcher[*roomin.GetRoomRequest, *roomout.GetRoomResponse]
	listRoom                 cqrs.Dispatcher[*roomin.ListRoomsRequest, *roomout.ListRoomsResponse]
	createDirectConversation cqrs.Dispatcher[*roomin.CreateDirectConversationRequest, *roomout.ChatConversationResponse]
	createGroupChat          cqrs.Dispatcher[*roomin.CreateGroupChatRequest, *roomout.ChatConversationResponse]
	updateGroupChat          cqrs.Dispatcher[*roomin.UpdateGroupChatRequest, *roomout.ChatConversationResponse]
	addChatMember            cqrs.Dispatcher[*roomin.AddChatMemberRequest, *roomout.ChatConversationResponse]
	removeChatMember         cqrs.Dispatcher[*roomin.RemoveChatMemberRequest, *roomout.ChatConversationResponse]
	pinChatMessage           cqrs.Dispatcher[*roomin.PinChatMessageRequest, *roomout.ChatConversationResponse]
	listChatConversations    cqrs.Dispatcher[*roomin.ListChatConversationsRequest, []*roomout.ChatConversationResponse]
	getChatConversation      cqrs.Dispatcher[*roomin.GetChatConversationRequest, *roomout.ChatConversationResponse]
	listChatMessages         cqrs.Dispatcher[*roomin.ListChatMessagesRequest, []*roomout.ChatMessageResponse]
	sendChatMessage          cqrs.Dispatcher[*roomin.SendChatMessageRequest, *roomout.ChatMessageResponse]
	editChatMessage          cqrs.Dispatcher[*roomin.EditChatMessageRequest, *roomout.ChatMessageResponse]
	deleteChatMessage        cqrs.Dispatcher[*roomin.DeleteChatMessageRequest, *roomout.DeleteChatMessageResponse]
	forwardChatMessage       cqrs.Dispatcher[*roomin.ForwardChatMessageRequest, *roomout.ChatMessageResponse]
	markChatMessageStatus    cqrs.Dispatcher[*roomin.MarkChatMessageStatusRequest, *roomout.MarkChatMessageStatusResponse]
	getChatPresence          cqrs.Dispatcher[*roomin.GetChatPresenceRequest, *roomout.ChatPresenceResponse]
	roomHub                  roomsocket.IHub
}

func NewHTTPServer(
	createRoom cqrs.Dispatcher[*roomin.CreateRoomRequest, *roomout.CreateRoomResponse],
	updateRoom cqrs.Dispatcher[*roomin.UpdateRoomRequest, *roomout.UpdateRoomResponse],
	deleteRoom cqrs.Dispatcher[*roomin.DeleteRoomRequest, *roomout.DeleteRoomResponse],
	getRoom cqrs.Dispatcher[*roomin.GetRoomRequest, *roomout.GetRoomResponse],
	listRoom cqrs.Dispatcher[*roomin.ListRoomsRequest, *roomout.ListRoomsResponse],
	createDirectConversation cqrs.Dispatcher[*roomin.CreateDirectConversationRequest, *roomout.ChatConversationResponse],
	createGroupChat cqrs.Dispatcher[*roomin.CreateGroupChatRequest, *roomout.ChatConversationResponse],
	updateGroupChat cqrs.Dispatcher[*roomin.UpdateGroupChatRequest, *roomout.ChatConversationResponse],
	addChatMember cqrs.Dispatcher[*roomin.AddChatMemberRequest, *roomout.ChatConversationResponse],
	removeChatMember cqrs.Dispatcher[*roomin.RemoveChatMemberRequest, *roomout.ChatConversationResponse],
	pinChatMessage cqrs.Dispatcher[*roomin.PinChatMessageRequest, *roomout.ChatConversationResponse],
	listChatConversations cqrs.Dispatcher[*roomin.ListChatConversationsRequest, []*roomout.ChatConversationResponse],
	getChatConversation cqrs.Dispatcher[*roomin.GetChatConversationRequest, *roomout.ChatConversationResponse],
	listChatMessages cqrs.Dispatcher[*roomin.ListChatMessagesRequest, []*roomout.ChatMessageResponse],
	sendChatMessage cqrs.Dispatcher[*roomin.SendChatMessageRequest, *roomout.ChatMessageResponse],
	editChatMessage cqrs.Dispatcher[*roomin.EditChatMessageRequest, *roomout.ChatMessageResponse],
	deleteChatMessage cqrs.Dispatcher[*roomin.DeleteChatMessageRequest, *roomout.DeleteChatMessageResponse],
	forwardChatMessage cqrs.Dispatcher[*roomin.ForwardChatMessageRequest, *roomout.ChatMessageResponse],
	markChatMessageStatus cqrs.Dispatcher[*roomin.MarkChatMessageStatusRequest, *roomout.MarkChatMessageStatusResponse],
	getChatPresence cqrs.Dispatcher[*roomin.GetChatPresenceRequest, *roomout.ChatPresenceResponse],
	roomHub roomsocket.IHub,
) (infrahttp.HTTPServer, error) {
	if roomHub == nil {
		return nil, stackerr.Error(fmt.Errorf("room hub can not be nil"))
	}

	return &roomServer{
		createRoom:               createRoom,
		updateRoom:               updateRoom,
		deleteRoom:               deleteRoom,
		getRoom:                  getRoom,
		listRoom:                 listRoom,
		createDirectConversation: createDirectConversation,
		createGroupChat:          createGroupChat,
		updateGroupChat:          updateGroupChat,
		addChatMember:            addChatMember,
		removeChatMember:         removeChatMember,
		pinChatMessage:           pinChatMessage,
		listChatConversations:    listChatConversations,
		getChatConversation:      getChatConversation,
		listChatMessages:         listChatMessages,
		sendChatMessage:          sendChatMessage,
		editChatMessage:          editChatMessage,
		deleteChatMessage:        deleteChatMessage,
		forwardChatMessage:       forwardChatMessage,
		markChatMessageStatus:    markChatMessageStatus,
		getChatPresence:          getChatPresence,
		roomHub:                  roomHub,
	}, nil
}

func (s *roomServer) RegisterPublicRoutes(_ *gin.RouterGroup) {
}

func (s *roomServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	roomhttp.RegisterPrivateRoutes(
		routes,
		s.createRoom,
		s.updateRoom,
		s.deleteRoom,
		s.getRoom,
		s.listRoom,
		s.createDirectConversation,
		s.createGroupChat,
		s.updateGroupChat,
		s.addChatMember,
		s.removeChatMember,
		s.pinChatMessage,
		s.listChatConversations,
		s.getChatConversation,
		s.listChatMessages,
		s.sendChatMessage,
		s.editChatMessage,
		s.deleteChatMessage,
		s.forwardChatMessage,
		s.markChatMessageStatus,
		s.getChatPresence,
		roomsocket.NewWSHandler(s.roomHub).Handle,
	)
}

func (s *roomServer) Stop(ctx context.Context) error {
	s.roomHub.Close(ctx)
	return nil
}
