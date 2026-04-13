package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/room/application/projection"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ChatQueryService struct {
	readRepos projection.QueryRepos
	redis     *redis.Client
}

func NewChatQueryService(readRepos projection.QueryRepos, redis *redis.Client) *ChatQueryService {
	return &ChatQueryService{
		readRepos: readRepos,
		redis:     redis,
	}
}

func (s *ChatQueryService) ListConversations(ctx context.Context, accountID string, query apptypes.ListConversationsQuery) ([]apptypes.ConversationResult, error) {
	log := logging.FromContext(ctx)

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	rooms, err := s.readRepos.RoomReadRepository().ListRoomsByAccount(ctx, accountID, utils.QueryOptions{
		Limit:          &limit,
		Offset:         &offset,
		OrderBy:        "updated_at",
		OrderDirection: "DESC",
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	out := make([]apptypes.ConversationResult, 0, len(rooms))
	for _, room := range rooms {
		if room == nil {
			continue
		}

		item, err := roomsupport.BuildConversationResult(ctx, s.readRepos, accountID, room, true)
		if err != nil {
			if errors.Is(err, roomsupport.ErrViewerNotMemberOfRoom) {
				log.Warnw(
					"skip inconsistent chat conversation projection",
					zap.String("room_id", room.ID),
					zap.String("account_id", accountID),
					zap.Error(err),
				)
				continue
			}
			return nil, stackErr.Error(err)
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *ChatQueryService) GetConversation(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationResult, error) {
	room, err := s.readRepos.RoomReadRepository().GetRoomByID(ctx, query.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return roomsupport.BuildConversationResult(ctx, s.readRepos, accountID, room, true)
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

	messages, err := s.readRepos.MessageReadRepository().ListMessages(ctx, accountID, query.RoomID, projection.MessageListOptions{
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
		item, err := roomsupport.BuildMessageResult(ctx, s.readRepos, accountID, message)
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

	room, err := s.readRepos.RoomReadRepository().GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if room == nil || !strings.EqualFold(strings.TrimSpace(room.RoomType), "group") {
		return nil, stackErr.Error(errors.New("mentions are supported only in group rooms"))
	}

	member, err := s.readRepos.RoomMemberReadRepository().GetRoomMemberByAccount(ctx, roomID, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if member == nil {
		return nil, stackErr.Error(errors.New("viewer is not a member of this room"))
	}

	candidates, err := s.readRepos.RoomMemberReadRepository().SearchMentionCandidates(
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
	exists, err := s.redis.Exists(ctx, chatPresenceKey(accountID)).Result()
	if err != nil {
		return nil, stackErr.Error(err)
	}
	status := "offline"
	if exists > 0 {
		status = "online"
	}
	return &apptypes.PresenceResult{AccountID: accountID, Status: status}, nil
}

func chatPresenceKey(accountID string) string {
	return "chat:presence:" + strings.TrimSpace(accountID)
}

func resolveMentionCandidateDisplayName(candidate *views.MentionCandidateView) string {
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
