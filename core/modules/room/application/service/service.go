package service

import (
	"context"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/room/application/projection"
	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/modules/room/types"
)

//go:generate mockgen -package=service -destination=service_mock.go -source=service.go
type Service interface {
	QueryService
	RealtimeService
}

type QueryService interface {
	ConversationQueryService
	MessageQueryService
	MentionQueryService
	PresenceQueryService
	RoomQueryService
}

type chatService struct {
	conversations ConversationQueryService
	messages      MessageQueryService
	mentions      MentionQueryService
	presence      PresenceQueryService
	realtime      RealtimeService
	room          RoomQueryService
}

func NewService(appCtx *appCtx.AppContext, readRepos projection.QueryRepos) Service {
	return &chatService{
		conversations: newConversationQueryService(readRepos),
		messages:      newMessageQueryService(readRepos),
		mentions:      newMentionQueryService(readRepos),
		presence:      newPresenceQueryService(appCtx),
		realtime:      newRealtimeService(appCtx),
		room:          newRoomQueryService(readRepos),
	}
}

func (s *chatService) ListConversations(ctx context.Context, accountID string, query apptypes.ListConversationsQuery) ([]apptypes.ConversationResult, error) {
	return s.conversations.ListConversations(ctx, accountID, query)
}

func (s *chatService) GetConversation(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationResult, error) {
	return s.conversations.GetConversation(ctx, accountID, query)
}

func (s *chatService) GetConversationMetadata(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationMetadataResult, error) {
	return s.conversations.GetConversationMetadata(ctx, accountID, query)
}

func (s *chatService) ListMessages(ctx context.Context, accountID string, query apptypes.ListMessagesQuery) ([]apptypes.MessageResult, error) {
	return s.messages.ListMessages(ctx, accountID, query)
}

func (s *chatService) SearchMentionCandidates(ctx context.Context, accountID string, query apptypes.SearchMentionCandidatesQuery) ([]apptypes.MentionCandidateResult, error) {
	return s.mentions.SearchMentionCandidates(ctx, accountID, query)
}

func (s *chatService) GetPresence(ctx context.Context, query apptypes.GetPresenceQuery) (*apptypes.PresenceResult, error) {
	return s.presence.GetPresence(ctx, query)
}

func (s *chatService) EmitMessage(ctx context.Context, message types.MessagePayload) error {
	return s.realtime.EmitMessage(ctx, message)
}

func (s *chatService) GetRoom(ctx context.Context, query apptypes.GetRoomQuery) (*apptypes.RoomResult, error) {
	return s.room.GetRoom(ctx, query)
}

func (s *chatService) ListRooms(ctx context.Context, query apptypes.ListRoomsQuery) (*apptypes.ListRoomsResult, error) {
	return s.room.ListRooms(ctx, query)
}
