package service

import (
	"context"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/utils"
)

type RoomQueryService struct {
	repos repos.QueryRepos
}

func NewRoomQueryService(repos repos.QueryRepos) *RoomQueryService {
	return &RoomQueryService{repos: repos}
}

func (s *RoomQueryService) GetRoom(ctx context.Context, query apptypes.GetRoomQuery) (*apptypes.RoomResult, error) {
	room, err := s.repos.RoomReadRepository().GetRoomByID(ctx, query.ID)
	if err != nil {
		return nil, err
	}
	return buildRoomResult(room), nil
}

func (s *RoomQueryService) ListRooms(ctx context.Context, query apptypes.ListRoomsQuery) (*apptypes.ListRoomsResult, error) {
	page := query.Page
	if page < 0 {
		page = 0
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	rooms, err := s.repos.RoomReadRepository().ListRooms(ctx, utils.QueryOptions{
		Offset:         &page,
		Limit:          &limit,
		OrderBy:        "updated_at",
		OrderDirection: "DESC",
	})
	if err != nil {
		return nil, err
	}

	result := &apptypes.ListRoomsResult{
		Page:  page,
		Limit: limit,
		Rooms: make([]apptypes.RoomResult, 0, len(rooms)),
	}
	for _, room := range rooms {
		roomResult := buildRoomResult(room)
		if roomResult != nil {
			result.Rooms = append(result.Rooms, *roomResult)
		}
	}

	return result, nil
}
