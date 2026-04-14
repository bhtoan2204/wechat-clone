package repos

import (
	"context"
	"go-socket/core/modules/room/domain/entity"
)

//go:generate mockgen -package=repos -destination=message_repo_mock.go -source=message_repo.go
type MessageRepository interface {
	CreateMessage(ctx context.Context, message *entity.MessageEntity) error
	UpdateMessage(ctx context.Context, message *entity.MessageEntity) error
	GetMessageByID(ctx context.Context, id string) (*entity.MessageEntity, error)
	GetLastMessageByRoomID(ctx context.Context, roomID string) (*entity.MessageEntity, error)
}
