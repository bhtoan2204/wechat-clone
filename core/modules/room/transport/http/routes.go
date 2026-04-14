// CODE_GENERATOR - do not edit: routing
package http

import (
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/transport/http/handler"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(_ *gin.RouterGroup) {}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
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
	searchChatMentions cqrs.Dispatcher[*in.SearchChatMentionsRequest, []*out.ChatMentionCandidateResponse],
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageResponse],
	editChatMessage cqrs.Dispatcher[*in.EditChatMessageRequest, *out.ChatMessageResponse],
	deleteChatMessage cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse],
	forwardChatMessage cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageResponse],
	markChatMessageStatus cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.MarkChatMessageStatusResponse],
	addChatMember cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatConversationResponse],
	removeChatMember cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatConversationResponse],
	pinChatMessage cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatConversationResponse],
	getChatPresence cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse],
) {
	routes.POST("/chat/room/create", httpx.Wrap(handler.NewCreateRoomHandler(createRoom)))
	routes.GET("/chat/room/list", httpx.Wrap(handler.NewListRoomsHandler(listRooms)))
	routes.GET("/chat/room/get", httpx.Wrap(handler.NewGetRoomHandler(getRoom)))
	routes.PUT("/chat/room/update", httpx.Wrap(handler.NewUpdateRoomHandler(updateRoom)))
	routes.DELETE("/chat/room/delete", httpx.Wrap(handler.NewDeleteRoomHandler(deleteRoom)))
	routes.POST("/chat/direct", httpx.Wrap(handler.NewCreateDirectConversationHandler(createDirectConversation)))
	routes.POST("/chat/groups", httpx.Wrap(handler.NewCreateGroupChatHandler(createGroupChat)))
	routes.PATCH("/chat/groups/:room_id", httpx.Wrap(handler.NewUpdateGroupChatHandler(updateGroupChat)))
	routes.GET("/chat/conversations", httpx.Wrap(handler.NewListChatConversationsHandler(listChatConversations)))
	routes.GET("/chat/conversations/:room_id", httpx.Wrap(handler.NewGetChatConversationHandler(getChatConversation)))
	routes.GET("/chat/conversations/:room_id/messages", httpx.Wrap(handler.NewListChatMessagesHandler(listChatMessages)))
	routes.GET("/chat/rooms/:room_id/mentions/search", httpx.Wrap(handler.NewSearchChatMentionsHandler(searchChatMentions)))
	routes.POST("/chat/messages", httpx.Wrap(handler.NewSendChatMessageHandler(sendChatMessage)))
	routes.PATCH("/chat/messages/:message_id", httpx.Wrap(handler.NewEditChatMessageHandler(editChatMessage)))
	routes.DELETE("/chat/messages/:message_id", httpx.Wrap(handler.NewDeleteChatMessageHandler(deleteChatMessage)))
	routes.POST("/chat/messages/:message_id/forward", httpx.Wrap(handler.NewForwardChatMessageHandler(forwardChatMessage)))
	routes.POST("/chat/messages/:message_id/status", httpx.Wrap(handler.NewMarkChatMessageStatusHandler(markChatMessageStatus)))
	routes.POST("/chat/rooms/:room_id/members", httpx.Wrap(handler.NewAddChatMemberHandler(addChatMember)))
	routes.DELETE("/chat/rooms/:room_id/members/:account_id", httpx.Wrap(handler.NewRemoveChatMemberHandler(removeChatMember)))
	routes.POST("/chat/rooms/:room_id/pin", httpx.Wrap(handler.NewPinChatMessageHandler(pinChatMessage)))
	routes.GET("/chat/presence/:account_id", httpx.Wrap(handler.NewGetChatPresenceHandler(getChatPresence)))
}
