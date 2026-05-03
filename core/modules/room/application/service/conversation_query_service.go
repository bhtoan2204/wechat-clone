package service

import (
	"context"
	"errors"
	"sync"

	"wechat-clone/core/modules/room/application/projection"
	roomsupport "wechat-clone/core/modules/room/application/support"
	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/modules/room/infra/projection/cassandra/views"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const listConversationBuildConcurrency = 8

type ConversationQueryService interface {
	ListConversations(ctx context.Context, accountID string, query apptypes.ListConversationsQuery) ([]apptypes.ConversationResult, error)
	GetConversation(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationResult, error)
	GetConversationMetadata(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationMetadataResult, error)
}

type conversationQueryService struct {
	readRepos projection.QueryRepos
}

func newConversationQueryService(readRepos projection.QueryRepos) ConversationQueryService {
	return &conversationQueryService{readRepos: readRepos}
}

func (s *conversationQueryService) ListConversations(ctx context.Context, accountID string, query apptypes.ListConversationsQuery) ([]apptypes.ConversationResult, error) {
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

	results, skippedRoomIDs, err := s.buildConversationResultsConcurrently(ctx, accountID, rooms)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	for _, roomID := range skippedRoomIDs {
		log.Warnw(
			"skip inconsistent chat conversation projection",
			zap.String("room_id", roomID),
			zap.String("account_id", accountID),
			zap.Error(roomsupport.ErrViewerNotMemberOfRoom),
		)
	}

	return results, nil
}

func (s *conversationQueryService) buildConversationResultsConcurrently(
	ctx context.Context,
	accountID string,
	rooms []*views.RoomView,
) ([]apptypes.ConversationResult, []string, error) {
	results := make([]*apptypes.ConversationResult, len(rooms))
	skippedRoomIDs := make([]string, 0)
	var skippedMu sync.Mutex

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(listConversationBuildConcurrency)
	for index, room := range rooms {
		index, room := index, room
		if room == nil {
			continue
		}

		eg.Go(func() error {
			item, err := roomsupport.BuildConversationResult(egCtx, s.readRepos, accountID, room, true)
			if err != nil {
				if errors.Is(err, roomsupport.ErrViewerNotMemberOfRoom) {
					skippedMu.Lock()
					skippedRoomIDs = append(skippedRoomIDs, room.ID)
					skippedMu.Unlock()
					return nil
				}
				return stackErr.Error(err)
			}
			if item != nil {
				copyItem := *item
				results[index] = &copyItem
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, nil, stackErr.Error(err)
	}

	out := make([]apptypes.ConversationResult, 0, len(results))
	for _, item := range results {
		if item == nil {
			continue
		}
		out = append(out, *item)
	}
	return out, skippedRoomIDs, nil
}

func (s *conversationQueryService) GetConversation(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationResult, error) {
	room, err := s.readRepos.RoomReadRepository().GetRoomByID(ctx, query.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return roomsupport.BuildConversationResult(ctx, s.readRepos, accountID, room, true)
}

func (s *conversationQueryService) GetConversationMetadata(ctx context.Context, accountID string, query apptypes.GetConversationQuery) (*apptypes.ConversationMetadataResult, error) {
	room, err := s.readRepos.RoomReadRepository().GetRoomByID(ctx, query.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return roomsupport.BuildConversationMetadataResult(ctx, s.readRepos, accountID, room)
}
