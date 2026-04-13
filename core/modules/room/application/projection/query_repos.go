package projection

import (
	"context"
	"time"

	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/utils"
)

type QueryRepos interface {
	RoomReadRepository() RoomReadRepository
	MessageReadRepository() MessageReadRepository
	RoomMemberReadRepository() RoomMemberReadRepository
}

type MessageListOptions struct {
	Limit     int
	BeforeID  string
	BeforeAt  *time.Time
	Ascending bool
}

type RoomReadRepository interface {
	ListRooms(ctx context.Context, options utils.QueryOptions) ([]*views.RoomView, error)
	ListRoomsByAccount(ctx context.Context, accountID string, options utils.QueryOptions) ([]*views.RoomView, error)
	GetRoomByID(ctx context.Context, id string) (*views.RoomView, error)
}

type MessageReadRepository interface {
	GetMessageByID(ctx context.Context, id string) (*views.MessageView, error)
	GetLastMessage(ctx context.Context, roomID string) (*views.MessageView, error)
	ListMessages(ctx context.Context, accountID, roomID string, options MessageListOptions) ([]*views.MessageView, error)
	GetMessageReceipt(ctx context.Context, messageID, accountID string) (string, *time.Time, *time.Time, error)
	CountMessageReceiptsByStatus(ctx context.Context, messageID, status string) (int64, error)
	CountUnreadMessages(ctx context.Context, roomID, accountID string, lastReadAt *time.Time) (int64, error)
}

type RoomMemberReadRepository interface {
	ListRoomMembers(ctx context.Context, roomID string) ([]*views.RoomMemberView, error)
	GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*views.RoomMemberView, error)
	SearchMentionCandidates(ctx context.Context, roomID, keyword, excludeAccountID string, limit int) ([]*views.MentionCandidateView, error)
}
