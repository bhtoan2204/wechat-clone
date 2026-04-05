package repos

import (
	"context"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/shared/utils"
)

type RoomRepository interface {
	CreateRoom(ctx context.Context, room *entity.Room) error
	ListRooms(ctx context.Context, options utils.QueryOptions) ([]*entity.Room, error)
	GetRoomByID(ctx context.Context, id string) (*entity.Room, error)
	GetRoomByDirectKey(ctx context.Context, directKey string) (*entity.Room, error)
	UpdateRoom(ctx context.Context, room *entity.Room) error
	DeleteRoom(ctx context.Context, id string) error
}
