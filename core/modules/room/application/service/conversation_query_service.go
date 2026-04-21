package service

import (
	"context"
	"errors"

	"wechat-clone/core/modules/room/application/projection"
	roomsupport "wechat-clone/core/modules/room/application/support"
	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"go.uber.org/zap"
)

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
