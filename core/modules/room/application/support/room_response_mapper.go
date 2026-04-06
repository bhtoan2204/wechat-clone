package support

import (
	"go-socket/core/modules/room/application/dto/out"
	apptypes "go-socket/core/modules/room/application/types"
)

func ToGetRoomResponse(room *apptypes.RoomResult) *out.GetRoomResponse {
	if room == nil {
		return nil
	}

	return &out.GetRoomResponse{
		ID:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		RoomType:    room.RoomType,
		OwnerID:     room.OwnerID,
		CreatedAt:   room.CreatedAt,
		UpdatedAt:   room.UpdatedAt,
	}
}

func ToListRoomsResponse(res *apptypes.ListRoomsResult) *out.ListRoomsResponse {
	if res == nil {
		return nil
	}

	rooms := make([]out.RoomResponse, 0, len(res.Rooms))
	for _, room := range res.Rooms {
		rooms = append(rooms, out.RoomResponse{
			ID:          room.ID,
			Name:        room.Name,
			Description: room.Description,
			RoomType:    room.RoomType,
			OwnerID:     room.OwnerID,
			CreatedAt:   room.CreatedAt,
			UpdatedAt:   room.UpdatedAt,
		})
	}

	return &out.ListRoomsResponse{
		Page:  res.Page,
		Limit: res.Limit,
		Rooms: rooms,
	}
}
