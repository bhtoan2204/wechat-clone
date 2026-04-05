package service

import (
	"context"
	"strings"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/utils"

	"github.com/redis/go-redis/v9"
)

type ChatQueryService struct {
	repos repos.QueryRepos
	redis *redis.Client
}

func NewChatQueryService(repos repos.QueryRepos, redis *redis.Client) *ChatQueryService {
	return &ChatQueryService{repos: repos, redis: redis}
}

func (s *ChatQueryService) ListConversations(ctx context.Context, accountID string, query apptypes.ListConversationsQuery) ([]apptypes.ConversationResult, error) {
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	rooms, err := s.repos.RoomReadRepository().ListRoomsByAccount(ctx, accountID, utils.QueryOptions{
		Limit:          &limit,
		Offset:         &offset,
		OrderBy:        "rr.updated_at",
		OrderDirection: "DESC",
	})
	if err != nil {
		return nil, err
	}

	out := make([]apptypes.ConversationResult, 0, len(rooms))
	for _, room := range rooms {
		item, err := buildConversationResult(ctx, s.repos, accountID, room, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *ChatQueryService) GetConversation(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationResult, error) {
	room, err := s.repos.RoomReadRepository().GetRoomByID(ctx, query.RoomID)
	if err != nil {
		return nil, err
	}
	return buildConversationResult(ctx, s.repos, accountID, room, true)
}

func (s *ChatQueryService) ListMessages(ctx context.Context, accountID string, query apptypes.ListMessagesQuery) ([]apptypes.MessageResult, error) {
	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var beforeAt *time.Time
	if strings.TrimSpace(query.BeforeAt) != "" {
		if parsed, err := time.Parse(time.RFC3339, query.BeforeAt); err == nil {
			beforeAt = &parsed
		}
	}

	messages, err := s.repos.MessageReadRepository().ListMessages(ctx, accountID, query.RoomID, repos.MessageListOptions{
		Limit:     limit,
		BeforeID:  query.BeforeID,
		BeforeAt:  beforeAt,
		Ascending: query.Ascending,
	})
	if err != nil {
		return nil, err
	}

	out := make([]apptypes.MessageResult, 0, len(messages))
	for _, message := range messages {
		item, err := buildMessageResult(ctx, s.repos, accountID, message)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *ChatQueryService) GetPresence(ctx context.Context, query apptypes.GetPresenceQuery) (*apptypes.PresenceResult, error) {
	accountID := query.AccountID
	if s.redis == nil {
		return &apptypes.PresenceResult{AccountID: accountID, Status: "offline"}, nil
	}
	exists, err := s.redis.Exists(ctx, presenceKey(accountID)).Result()
	if err != nil {
		return nil, err
	}
	status := "offline"
	if exists > 0 {
		status = "online"
	}
	return &apptypes.PresenceResult{AccountID: accountID, Status: status}, nil
}
