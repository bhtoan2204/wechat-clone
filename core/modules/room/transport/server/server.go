// CODE_GENERATOR: registry
package server

import (
	"context"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	roomhttp "wechat-clone/core/modules/room/transport/http"
	roomsocket "wechat-clone/core/modules/room/transport/websocket"
	"wechat-clone/core/shared/pkg/cqrs"
	infrahttp "wechat-clone/core/shared/transport/http"

	"github.com/gin-gonic/gin"
)

type roomHTTPServer struct {
	createDirectConversation      cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatRoomCommandResponse]
	createGroupChat               cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatRoomCommandResponse]
	updateGroupChat               cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatRoomCommandResponse]
	listChatConversations         cqrs.Dispatcher[*in.ListChatConversationsRequest, []*out.ChatConversationResponse]
	getChatConversation           cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse]
	getChatConversationMetadata   cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationMetadataResponse]
	listChatMessages              cqrs.Dispatcher[*in.ListChatMessagesRequest, []*out.ChatMessageResponse]
	searchChatMentions            cqrs.Dispatcher[*in.SearchChatMentionsRequest, []*out.ChatMentionCandidateResponse]
	createChatMessagePresignedURL cqrs.Dispatcher[*in.CreateChatMessagePresignedURLRequest, *out.CreateChatMessagePresignedURLResponse]
	getChatMessageMedia           cqrs.Dispatcher[*in.GetChatMessageMediaRequest, *out.GetChatMessageMediaResponse]
	sendChatMessage               cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageCommandResponse]
	toggleChatMessageReaction     cqrs.Dispatcher[*in.ToggleChatMessageReactionRequest, *out.ChatMessageCommandResponse]
	editChatMessage               cqrs.Dispatcher[*in.EditChatMessageRequest, *out.ChatMessageCommandResponse]
	deleteChatMessage             cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.ChatMessageCommandResponse]
	forwardChatMessage            cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageCommandResponse]
	markChatMessageStatus         cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.ChatMessageCommandResponse]
	addChatMember                 cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatRoomCommandResponse]
	removeChatMember              cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatRoomCommandResponse]
	pinChatMessage                cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatRoomCommandResponse]
	getChatPresence               cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse]
	socketHandler                 gin.HandlerFunc
	socketStopper                 func(context.Context)
}

func NewHTTPServer(
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
	socketHandler gin.HandlerFunc,
	socketStopper func(context.Context),
) (infrahttp.HTTPServer, error) {
	return &roomHTTPServer{
		createDirectConversation:      createDirectConversation,
		createGroupChat:               createGroupChat,
		updateGroupChat:               updateGroupChat,
		listChatConversations:         listChatConversations,
		getChatConversation:           getChatConversation,
		getChatConversationMetadata:   getChatConversationMetadata,
		listChatMessages:              listChatMessages,
		searchChatMentions:            searchChatMentions,
		createChatMessagePresignedURL: createChatMessagePresignedURL,
		getChatMessageMedia:           getChatMessageMedia,
		sendChatMessage:               sendChatMessage,
		toggleChatMessageReaction:     toggleChatMessageReaction,
		editChatMessage:               editChatMessage,
		deleteChatMessage:             deleteChatMessage,
		forwardChatMessage:            forwardChatMessage,
		markChatMessageStatus:         markChatMessageStatus,
		addChatMember:                 addChatMember,
		removeChatMember:              removeChatMember,
		pinChatMessage:                pinChatMessage,
		getChatPresence:               getChatPresence,
		socketHandler:                 socketHandler,
		socketStopper:                 socketStopper,
	}, nil
}

func (s *roomHTTPServer) RegisterPublicRoutes(routes *gin.RouterGroup) {
	roomhttp.RegisterPublicRoutes(routes)
}

func (s *roomHTTPServer) RegisterPrivateRoutes(routes *gin.RouterGroup) {
	roomhttp.RegisterPrivateRoutes(routes, s.createDirectConversation, s.createGroupChat, s.updateGroupChat, s.listChatConversations, s.getChatConversation, s.getChatConversationMetadata, s.listChatMessages, s.searchChatMentions, s.createChatMessagePresignedURL, s.getChatMessageMedia, s.sendChatMessage, s.toggleChatMessageReaction, s.editChatMessage, s.deleteChatMessage, s.forwardChatMessage, s.markChatMessageStatus, s.addChatMember, s.removeChatMember, s.pinChatMessage, s.getChatPresence)
}

func (s *roomHTTPServer) RegisterSocketRoutes(routes *gin.RouterGroup) {
	roomsocket.RegisterPrivateRoutes(routes, s.socketHandler)
}

func (s *roomHTTPServer) Stop(ctx context.Context) error {
	if s.socketStopper != nil {
		s.socketStopper(ctx)
	}
	return nil
}
