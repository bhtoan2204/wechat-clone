package repository

import (
	"context"

	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/shared/utils"
)

type roomStore interface {
	CreateRoom(ctx context.Context, room *entity.Room) error
	ListRooms(ctx context.Context, options utils.QueryOptions) ([]*entity.Room, error)
	GetRoomByID(ctx context.Context, id string) (*entity.Room, error)
	GetRoomByDirectKey(ctx context.Context, directKey string) (*entity.Room, error)
	UpdateRoom(ctx context.Context, room *entity.Room) error
	DeleteRoom(ctx context.Context, id string) error
}

type messageStore interface {
	CreateMessage(ctx context.Context, message *entity.MessageEntity) error
	UpdateMessage(ctx context.Context, message *entity.MessageEntity) error
	GetMessageByID(ctx context.Context, id string) (*entity.MessageEntity, error)
	GetLastMessageByRoomID(ctx context.Context, roomID string) (*entity.MessageEntity, error)
}

type roomMemberStore interface {
	CreateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error
	DeleteRoomMember(ctx context.Context, roomID, accountID string) error
	GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error)
	ListRoomMembers(ctx context.Context, roomID string) ([]*entity.RoomMemberEntity, error)
	UpdateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error
}

type accountProjectionStore interface {
	ProjectAccount(context.Context, *entity.AccountEntity) error
	ListByAccountIDs(ctx context.Context, accountIDs []string) ([]*entity.AccountEntity, error)
}
