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
	createDirectConversation      cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatConversationResponse]
	createGroupChat               cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatConversationResponse]
	updateGroupChat               cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatConversationResponse]
	listChatConversations         cqrs.Dispatcher[*in.ListChatConversationsRequest, []*out.ChatConversationResponse]
	getChatConversation           cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse]
	listChatMessages              cqrs.Dispatcher[*in.ListChatMessagesRequest, []*out.ChatMessageResponse]
	searchChatMentions            cqrs.Dispatcher[*in.SearchChatMentionsRequest, []*out.ChatMentionCandidateResponse]
	createChatMessagePresignedURL cqrs.Dispatcher[*in.CreateChatMessagePresignedURLRequest, *out.CreateChatMessagePresignedURLResponse]
	getChatMessageMedia           cqrs.Dispatcher[*in.GetChatMessageMediaRequest, *out.GetChatMessageMediaResponse]
	sendChatMessage               cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageResponse]
	toggleChatMessageReaction     cqrs.Dispatcher[*in.ToggleChatMessageReactionRequest, *out.ChatMessageResponse]
	editChatMessage               cqrs.Dispatcher[*in.EditChatMessageRequest, *out.ChatMessageResponse]
	deleteChatMessage             cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse]
	forwardChatMessage            cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageResponse]
	markChatMessageStatus         cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.MarkChatMessageStatusResponse]
	addChatMember                 cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatConversationResponse]
	removeChatMember              cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatConversationResponse]
	pinChatMessage                cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatConversationResponse]
	getChatPresence               cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse]
	socketHandler                 gin.HandlerFunc
	socketStopper                 func(context.Context)
}

func NewHTTPServer(
	createDirectConversation cqrs.Dispatcher[*in.CreateDirectConversationRequest, *out.ChatConversationResponse],
	createGroupChat cqrs.Dispatcher[*in.CreateGroupChatRequest, *out.ChatConversationResponse],
	updateGroupChat cqrs.Dispatcher[*in.UpdateGroupChatRequest, *out.ChatConversationResponse],
	listChatConversations cqrs.Dispatcher[*in.ListChatConversationsRequest, []*out.ChatConversationResponse],
	getChatConversation cqrs.Dispatcher[*in.GetChatConversationRequest, *out.ChatConversationResponse],
	listChatMessages cqrs.Dispatcher[*in.ListChatMessagesRequest, []*out.ChatMessageResponse],
	searchChatMentions cqrs.Dispatcher[*in.SearchChatMentionsRequest, []*out.ChatMentionCandidateResponse],
	createChatMessagePresignedURL cqrs.Dispatcher[*in.CreateChatMessagePresignedURLRequest, *out.CreateChatMessagePresignedURLResponse],
	getChatMessageMedia cqrs.Dispatcher[*in.GetChatMessageMediaRequest, *out.GetChatMessageMediaResponse],
	sendChatMessage cqrs.Dispatcher[*in.SendChatMessageRequest, *out.ChatMessageResponse],
	toggleChatMessageReaction cqrs.Dispatcher[*in.ToggleChatMessageReactionRequest, *out.ChatMessageResponse],
	editChatMessage cqrs.Dispatcher[*in.EditChatMessageRequest, *out.ChatMessageResponse],
	deleteChatMessage cqrs.Dispatcher[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse],
	forwardChatMessage cqrs.Dispatcher[*in.ForwardChatMessageRequest, *out.ChatMessageResponse],
	markChatMessageStatus cqrs.Dispatcher[*in.MarkChatMessageStatusRequest, *out.MarkChatMessageStatusResponse],
	addChatMember cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatConversationResponse],
	removeChatMember cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatConversationResponse],
	pinChatMessage cqrs.Dispatcher[*in.PinChatMessageRequest, *out.ChatConversationResponse],
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
	roomhttp.RegisterPrivateRoutes(routes, s.createDirectConversation, s.createGroupChat, s.updateGroupChat, s.listChatConversations, s.getChatConversation, s.listChatMessages, s.searchChatMentions, s.createChatMessagePresignedURL, s.getChatMessageMedia, s.sendChatMessage, s.toggleChatMessageReaction, s.editChatMessage, s.deleteChatMessage, s.forwardChatMessage, s.markChatMessageStatus, s.addChatMember, s.removeChatMember, s.pinChatMessage, s.getChatPresence)
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
