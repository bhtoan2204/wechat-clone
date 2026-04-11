package repos

import (
	"context"
	"time"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/shared/utils"
)

type Repos interface {
	RoomRepository() RoomRepository
	MessageRepository() MessageRepository
	RoomMemberRepository() RoomMemberRepository
	RoomOutboxEventsRepository() RoomOutboxEventsRepository

	RoomReadRepository() RoomReadRepository
	MessageReadRepository() MessageReadRepository
	RoomMemberReadRepository() RoomMemberReadRepository
	RoomAccountProjectionRepository() RoomAccountProjectionRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}

type QueryRepos interface {
	RoomReadRepository() RoomReadRepository
	MessageReadRepository() MessageReadRepository
	RoomMemberReadRepository() RoomMemberReadRepository
	RoomAccountProjectionRepository() RoomAccountProjectionRepository
}

type MessageListOptions struct {
	Limit     int
	BeforeID  string
	BeforeAt  *time.Time
	Ascending bool
}

type RoomReadRepository interface {
	UpsertRoom(ctx context.Context, room *entity.Room) error
	UpdateRoom(ctx context.Context, room *entity.Room) error
	DeleteRoom(ctx context.Context, id string) error
	ListRooms(ctx context.Context, options utils.QueryOptions) ([]*entity.Room, error)
	ListRoomsByAccount(ctx context.Context, accountID string, options utils.QueryOptions) ([]*entity.Room, error)
	GetRoomByID(ctx context.Context, id string) (*entity.Room, error)
	UpdateRoomStats(ctx context.Context, roomID string, memberCount int, lastMessage *entity.MessageEntity, updatedAt time.Time) error
	UpdatePinnedMessage(ctx context.Context, roomID, pinnedMessageID string, updatedAt time.Time) error
}

type MessageReadRepository interface {
	UpsertMessage(ctx context.Context, message *entity.MessageEntity) error
	GetMessageByID(ctx context.Context, id string) (*entity.MessageEntity, error)
	GetLastMessage(ctx context.Context, roomID string) (*entity.MessageEntity, error)
	ListMessages(ctx context.Context, accountID, roomID string, options MessageListOptions) ([]*entity.MessageEntity, error)
	UpsertMessageReceipt(ctx context.Context, messageID, accountID, status string, deliveredAt, seenAt *time.Time, createdAt, updatedAt time.Time) error
	GetMessageReceipt(ctx context.Context, messageID, accountID string) (string, *time.Time, *time.Time, error)
	CountMessageReceiptsByStatus(ctx context.Context, messageID, status string) (int64, error)
	UpsertMessageDeletion(ctx context.Context, messageID, accountID string, createdAt time.Time) error
	CountUnreadMessages(ctx context.Context, roomID, accountID string, lastReadAt *time.Time) (int64, error)
}

type RoomMemberReadRepository interface {
	UpsertRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error
	DeleteRoomMember(ctx context.Context, roomID, accountID string) error
	ListRoomMembers(ctx context.Context, roomID string) ([]*entity.RoomMemberEntity, error)
	GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error)
	SearchMentionCandidates(ctx context.Context, roomID, keyword, excludeAccountID string, limit int) ([]*entity.MentionCandidate, error)
}
