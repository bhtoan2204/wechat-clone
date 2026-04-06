// CODE_GENERATOR: registry
package server

import (
	"context"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomhttp "go-socket/core/modules/room/transport/http"
	"go-socket/core/shared/pkg/cqrs"
	infrahttp "go-socket/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type roomHTTPServer struct {
	createRoom               cqrs.Dispatcher[*in.CreateRoomRequest, *out.CreateRoomResponse]
	listRooms                cqrs.Dispatcher[*in.ListRoomsRequest, *out.ListRoomsResponse]
	getRoom                  cqrs.Dispatcher[*in.GetRoomRequest, *out.GetRoomResponse]
	updateRoom               cqrs.Dispatcher[*in.UpdateRoomRequest, *out.UpdateRoomResponse]
	deleteRoom               cqrs.Dispatcher[*in.DeleteRoomRequest, *out.DeleteRoomResponse]
	createDirectConversation cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatConversationResponse]
	createGroupChat          cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatConversationResponse]
	updateGroupChat          cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatConversationResponse]
	listChatConversations    cqrs.Dispatcher[*in.ListChatConversationsRequest, []*out.ChatConversationResponse]
	getChatConversation      cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse]
	listChatMessages         cqrs.Dispatcher[*in.ListChatMessagesRequest, []*out.ChatMessageResponse]
	sendChatMessage          cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageResponse]
	editChatMessage          cqrs.Dispatcher[*in.EditChatMessageRequest, *out.ChatMessageResponse]
	deleteChatMessage        cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse]
	forwardChatMessage       cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageResponse]
	markChatMessageStatus    cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.MarkChatMessageStatusResponse]
	addChatMember            cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatConversationResponse]
	removeChatMember         cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatConversationResponse]
	pinChatMessage           cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatConversationResponse]
	getChatPresence          cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse]
}

func NewHTTPServer(
	createRoom cqrs.Dispatcher[*in.CreateRoomRequest, *out.CreateRoomResponse],
	listRooms cqrs.Dispatcher[*in.ListRoomsRequest, *out.ListRoomsResponse],
	getRoom cqrs.Dispatcher[*in.GetRoomRequest, *out.GetRoomResponse],
	updateRoom cqrs.Dispatcher[*in.UpdateRoomRequest, *out.UpdateRoomResponse],
	deleteRoom cqrs.Dispatcher[*in.DeleteRoomRequest, *out.DeleteRoomResponse],
	createDirectConversation cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatConversationResponse],
	createGroupChat cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatConversationResponse],
	updateGroupChat cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatConversationResponse],
	listChatConversations cqrs.Dispatcher[*in.ListChatConversationsRequest, []*out.ChatConversationResponse],
	getChatConversation cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse],
	listChatMessages cqrs.Dispatcher[*in.ListChatMessagesRequest, []*out.ChatMessageResponse],
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageResponse],
	editChatMessage cqrs.Dispatcher[*in.EditChatMessageRequest, *out.ChatMessageResponse],
	deleteChatMessage cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse],
	forwardChatMessage cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageResponse],
	markChatMessageStatus cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.MarkChatMessageStatusResponse],
	addChatMember cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatConversationResponse],
	removeChatMember cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatConversationResponse],
	pinChatMessage cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatConversationResponse],
	getChatPresence cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse],
) (infrahttp.HTTPServer, error) {
	return &roomHTTPServer{
		createRoom:               createRoom,
		listRooms:                listRooms,
		getRoom:                  getRoom,
		updateRoom:               updateRoom,
		deleteRoom:               deleteRoom,
		createDirectConversation: createDirectConversation,
		createGroupChat:          createGroupChat,
		updateGroupChat:          updateGroupChat,
		listChatConversations:    listChatConversations,
		getChatConversation:      getChatConversation,
		listChatMessages:         listChatMessages,
		sendChatMessage:          sendChatMessage,
		editChatMessage:          editChatMessage,
		deleteChatMessage:        deleteChatMessage,
		forwardChatMessage:       forwardChatMessage,
		markChatMessageStatus:    markChatMessageStatus,
		addChatMember:            addChatMember,
		removeChatMember:         removeChatMember,
		pinChatMessage:           pinChatMessage,
		getChatPresence:          getChatPresence,
	}, nil
}

func (s *roomHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	roomhttp.RegisterPublicRoutes(routes)
}

func (s *roomHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	roomhttp.RegisterPrivateRoutes(routes, s.createRoom, s.listRooms, s.getRoom, s.updateRoom, s.deleteRoom, s.createDirectConversation, s.createGroupChat, s.updateGroupChat, s.listChatConversations, s.getChatConversation, s.listChatMessages, s.sendChatMessage, s.editChatMessage, s.deleteChatMessage, s.forwardChatMessage, s.markChatMessageStatus, s.addChatMember, s.removeChatMember, s.pinChatMessage, s.getChatPresence)
}

func (s *roomHTTPServer) Stop(_ context.Context) error {
	return nil
}
