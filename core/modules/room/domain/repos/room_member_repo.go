package repos

import (
	"context"
	"go-socket/core/modules/room/domain/entity"
)

//go:generate mockgen -package=repos -destination=room_member_repo_mock.go -source=room_member_repo.go
type RoomMemberRepository interface {
	CreateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error
	DeleteRoomMember(ctx context.Context, roomID, accountID string) error
	GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error)
	ListRoomMembers(ctx context.Context, roomID string) ([]*entity.RoomMemberEntity, error)
	UpdateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error
}
