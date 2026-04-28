// CODE_GENERATOR - do not edit: routing
package http

import (
	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/modules/room/transport/http/handler"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/transport/httpx"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(_ *gin.RouterGroup) {}
func RegisterPrivateRoutes(
	routes *gin.RouterGroup,
	createDirectConversation cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatRoomCommandResponse],
	createGroupChat cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatRoomCommandResponse],
	updateGroupChat cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatRoomCommandResponse],
	listChatConversations cqrs.Dispatcher[*in.ListChatConversationsRequest, []*out.ChatConversationResponse],
	getChatConversation cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse],
	getChatConversationMetadata cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationMetadataResponse],
	listChatMessages cqrs.Dispatcher[*in.ListChatMessagesRequest, []*out.ChatMessageResponse],
	searchChatMentions cqrs.Dispatcher[*in.SearchChatMentionsRequest, []*out.ChatMentionCandidateResponse],
	createChatMessagePresignedURL cqrs.Dispatcher[*in.CreateChatMessagePresignedURLRequest, *out.CreateChatMessagePresignedURLResponse],
	getChatMessageMedia cqrs.Dispatcher[*in.GetChatMessageMediaRequest, *out.GetChatMessageMediaResponse],
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageCommandResponse],
	toggleChatMessageReaction cqrs.Dispatcher[*in.ToggleChatMessageReactionRequest, *out.ChatMessageCommandResponse],
	editChatMessage cqrs.Dispatcher[*in.EditChatMessageRequest, *out.ChatMessageCommandResponse],
	deleteChatMessage cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.ChatMessageCommandResponse],
	forwardChatMessage cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageCommandResponse],
	markChatMessageStatus cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.ChatMessageCommandResponse],
	addChatMember cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatRoomCommandResponse],
	removeChatMember cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatRoomCommandResponse],
	pinChatMessage cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatRoomCommandResponse],
	getChatPresence cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse],
) {
	routes.POST("/chat/direct", httpx.Wrap(handler.NewCreateDirectConversationHandler(createDirectConversation)))
	routes.POST("/chat/groups", httpx.Wrap(handler.NewCreateGroupChatHandler(createGroupChat)))
	routes.PATCH("/chat/groups/:room_id", httpx.Wrap(handler.NewUpdateGroupChatHandler(updateGroupChat)))
	routes.GET("/chat/conversations", httpx.Wrap(handler.NewListChatConversationsHandler(listChatConversations)))
	routes.GET("/chat/conversations/:room_id", httpx.Wrap(handler.NewGetChatConversationHandler(getChatConversation)))
	routes.GET("/chat/conversations/:room_id/metadata", httpx.Wrap(handler.NewGetChatConversationMetadataHandler(getChatConversationMetadata)))
	routes.GET("/chat/conversations/:room_id/messages", httpx.Wrap(handler.NewListChatMessagesHandler(listChatMessages)))
	routes.GET("/chat/rooms/:room_id/mentions/search", httpx.Wrap(handler.NewSearchChatMentionsHandler(searchChatMentions)))
	routes.POST("/chat/messages/presigned-url", httpx.Wrap(handler.NewCreateChatMessagePresignedURLHandler(createChatMessagePresignedURL)))
	routes.GET("/chat/messages/media", httpx.Wrap(handler.NewGetChatMessageMediaHandler(getChatMessageMedia)))
	routes.POST("/chat/messages", httpx.Wrap(handler.NewSendChatMessageHandler(sendChatMessage)))
	routes.POST("/chat/messages/:message_id/reactions", httpx.Wrap(handler.NewToggleChatMessageReactionHandler(toggleChatMessageReaction)))
	routes.PATCH("/chat/messages/:message_id", httpx.Wrap(handler.NewEditChatMessageHandler(editChatMessage)))
	routes.DELETE("/chat/messages/:message_id", httpx.Wrap(handler.NewDeleteChatMessageHandler(deleteChatMessage)))
	routes.POST("/chat/messages/:message_id/forward", httpx.Wrap(handler.NewForwardChatMessageHandler(forwardChatMessage)))
	routes.POST("/chat/messages/:message_id/status", httpx.Wrap(handler.NewMarkChatMessageStatusHandler(markChatMessageStatus)))
	routes.POST("/chat/rooms/:room_id/members", httpx.Wrap(handler.NewAddChatMemberHandler(addChatMember)))
	routes.DELETE("/chat/rooms/:room_id/members/:account_id", httpx.Wrap(handler.NewRemoveChatMemberHandler(removeChatMember)))
	routes.POST("/chat/rooms/:room_id/pin", httpx.Wrap(handler.NewPinChatMessageHandler(pinChatMessage)))
	routes.GET("/chat/presence/:account_id", httpx.Wrap(handler.NewGetChatPresenceHandler(getChatPresence)))
}
