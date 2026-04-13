package service

import (
	"context"

	"go-socket/core/modules/room/application/projection"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"
)

type RoomQueryService struct {
	readRepos projection.QueryRepos
}

func NewRoomQueryService(readRepos projection.QueryRepos) *RoomQueryService {
	return &RoomQueryService{readRepos: readRepos}
}

func (s *RoomQueryService) GetRoom(ctx context.Context, query apptypes.GetRoomQuery) (*apptypes.RoomResult, error) {
	room, err := s.readRepos.RoomReadRepository().GetRoomByID(ctx, query.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return roomsupport.BuildRoomResult(room), nil
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

	rooms, err := s.readRepos.RoomReadRepository().ListRooms(ctx, utils.QueryOptions{
		Offset:         &page,
		Limit:          &limit,
		OrderBy:        "updated_at",
		OrderDirection: "DESC",
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	result := &apptypes.ListRoomsResult{
		Page:  page,
		Limit: limit,
		Rooms: make([]apptypes.RoomResult, 0, len(rooms)),
	}
	for _, room := range rooms {
		roomResult := roomsupport.BuildRoomResult(room)
		if roomResult != nil {
			result.Rooms = append(result.Rooms, *roomResult)
		}
	}

	return result, nil
}
