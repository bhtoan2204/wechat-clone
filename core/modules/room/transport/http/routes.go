package http

import (
	roomin "go-socket/core/modules/room/application/dto/in"
	roomout "go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
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
	wsHandler gin.HandlerFunc,
) {
	routes.POST("/room/create", httpx.Wrap(handler.NewCreateRoomHandler(createRoom)))
	routes.GET("/room/list", httpx.Wrap(handler.NewListRoomsHandler(listRoom)))
	routes.GET("/room/get", httpx.Wrap(handler.NewGetRoomHandler(getRoom)))
	routes.PUT("/room/update", httpx.Wrap(handler.NewUpdateRoomHandler(updateRoom)))
	routes.DELETE("/room/delete", httpx.Wrap(handler.NewDeleteRoomHandler(deleteRoom)))
	routes.GET("/room/ws", wsHandler)

	routes.POST("/chat/direct", wrapChat(handler.NewCreateDirectConversationHandler(createDirectConversation)))
	routes.POST("/chat/groups", wrapChat(handler.NewCreateGroupChatHandler(createGroupChat)))
	routes.PATCH("/chat/groups/:room_id", wrapChat(handler.NewUpdateGroupChatHandler(updateGroupChat)))
	routes.GET("/chat/conversations", wrapChat(handler.NewListChatConversationsHandler(listChatConversations)))
	routes.GET("/chat/conversations/:room_id", wrapChat(handler.NewGetChatConversationHandler(getChatConversation)))
	routes.GET("/chat/conversations/:room_id/messages", wrapChat(handler.NewListChatMessagesHandler(listChatMessages)))
	routes.POST("/chat/messages", wrapChat(handler.NewSendChatMessageHandler(sendChatMessage)))
	routes.PATCH("/chat/messages/:message_id", wrapChat(handler.NewEditChatMessageHandler(editChatMessage)))
	routes.DELETE("/chat/messages/:message_id", wrapChat(handler.NewDeleteChatMessageHandler(deleteChatMessage)))
	routes.POST("/chat/messages/:message_id/forward", wrapChat(handler.NewForwardChatMessageHandler(forwardChatMessage)))
	routes.POST("/chat/messages/:message_id/status", wrapChat(handler.NewMarkChatMessageStatusHandler(markChatMessageStatus)))
	routes.POST("/chat/rooms/:room_id/members", wrapChat(handler.NewAddChatMemberHandler(addChatMember)))
	routes.DELETE("/chat/rooms/:room_id/members/:account_id", wrapChat(handler.NewRemoveChatMemberHandler(removeChatMember)))
	routes.POST("/chat/rooms/:room_id/pin", wrapChat(handler.NewPinChatMessageHandler(pinChatMessage)))
	routes.GET("/chat/presence/:account_id", wrapChat(handler.NewGetChatPresenceHandler(getChatPresence)))
}

func wrapChat(h interface {
	Handle(c *gin.Context) (interface{}, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "handler is nil"})
			return
		}
		data, err := h.Handle(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)
	}
}
