package repos

import (
	"context"
	"go-socket/core/modules/room/domain/entity"
)

type MessageRepository interface {
	CreateMessage(ctx context.Context, message *entity.MessageEntity) error
	UpdateMessage(ctx context.Context, message *entity.MessageEntity) error
	GetMessageByID(ctx context.Context, id string) (*entity.MessageEntity, error)
	GetLastMessageByRoomID(ctx context.Context, roomID string) (*entity.MessageEntity, error)
}
