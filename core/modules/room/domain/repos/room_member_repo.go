package repos

import (
	"context"
	"go-socket/core/modules/room/domain/entity"
)

type RoomMemberRepository interface {
	CreateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error
	DeleteRoomMember(ctx context.Context, roomID, accountID string) error
	GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error)
}
