package service

import (
	"context"
	"errors"
	"strings"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/stackErr"
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
		return nil, stackErr.Error(err)
	}

	out := make([]apptypes.ConversationResult, 0, len(rooms))
	for _, room := range rooms {
		item, err := buildConversationResult(ctx, s.repos, accountID, room, true)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *ChatQueryService) GetConversation(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationResult, error) {
	room, err := s.repos.RoomReadRepository().GetRoomByID(ctx, query.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
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
		return nil, stackErr.Error(err)
	}

	out := make([]apptypes.MessageResult, 0, len(messages))
	for _, message := range messages {
		item, err := buildMessageResult(ctx, s.repos, accountID, message)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *ChatQueryService) SearchMentionCandidates(ctx context.Context, accountID string, query apptypes.SearchMentionCandidatesQuery) ([]apptypes.MentionCandidateResult, error) {
	roomID := strings.TrimSpace(query.RoomID)
	if roomID == "" {
		return nil, stackErr.Error(errors.New("room_id is required"))
	}

	room, err := s.repos.RoomReadRepository().GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := room.RequireGroup(); err != nil {
		return nil, stackErr.Error(entity.ErrRoomMentionsRequireGroup)
	}

	member, err := s.repos.RoomMemberReadRepository().GetRoomMemberByAccount(ctx, roomID, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if member == nil {
		return nil, stackErr.Error(entity.ErrRoomMemberRequired)
	}

	candidates, err := s.repos.RoomMemberReadRepository().SearchMentionCandidates(
		ctx,
		roomID,
		query.Query,
		accountID,
		query.Limit,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	results := make([]apptypes.MentionCandidateResult, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		results = append(results, apptypes.MentionCandidateResult{
			AccountID:       candidate.AccountID,
			DisplayName:     resolveMentionCandidateDisplayName(candidate),
			Username:        strings.TrimSpace(candidate.Username),
			AvatarObjectKey: strings.TrimSpace(candidate.AvatarObjectKey),
		})
	}

	return results, nil
}

func (s *ChatQueryService) GetPresence(ctx context.Context, query apptypes.GetPresenceQuery) (*apptypes.PresenceResult, error) {
	accountID := query.AccountID
	if s.redis == nil {
		return &apptypes.PresenceResult{AccountID: accountID, Status: "offline"}, nil
	}
	exists, err := s.redis.Exists(ctx, presenceKey(accountID)).Result()
	if err != nil {
		return nil, stackErr.Error(err)
	}
	status := "offline"
	if exists > 0 {
		status = "online"
	}
	return &apptypes.PresenceResult{AccountID: accountID, Status: status}, nil
}

func resolveMentionCandidateDisplayName(candidate *entity.MentionCandidate) string {
	if candidate == nil {
		return ""
	}

	switch {
	case strings.TrimSpace(candidate.DisplayName) != "":
		return strings.TrimSpace(candidate.DisplayName)
	case strings.TrimSpace(candidate.Username) != "":
		return strings.TrimSpace(candidate.Username)
	default:
		return strings.TrimSpace(candidate.AccountID)
	}
}
