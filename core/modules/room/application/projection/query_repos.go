package projection

import (
	"context"
	"time"

	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/utils"
)

//go:generate mockgen -package=projection -destination=query_repos_mock.go -source=query_repos.go
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

type MessageReceiptLookup struct {
	MessageID string
	AccountID string
}

type MessageReceiptStatus struct {
	Status      string
	DeliveredAt *time.Time
	SeenAt      *time.Time
}

type MentionCandidateSearch struct {
	RoomID           string
	Keyword          string
	ExcludeAccountID string
	Limit            int
}

//go:generate mockgen -package=projection -destination=query_repos_mock.go -source=query_repos.go
type RoomReadRepository interface {
	ListRooms(ctx context.Context, options utils.QueryOptions) ([]*views.RoomView, error)
	ListRoomsByAccount(ctx context.Context, accountID string, options utils.QueryOptions) ([]*views.RoomView, error)
	GetRoomByID(ctx context.Context, id string) (*views.RoomView, error)
}

//go:generate mockgen -package=projection -destination=query_repos_mock.go -source=query_repos.go
type MessageReadRepository interface {
	GetMessageByID(ctx context.Context, id string) (*views.MessageView, error)
	GetLastMessage(ctx context.Context, roomID string) (*views.MessageView, error)
	ListMessages(ctx context.Context, accountID, roomID string, options MessageListOptions) ([]*views.MessageView, error)
	GetMessageReceipt(ctx context.Context, lookup MessageReceiptLookup) (*MessageReceiptStatus, error)
	CountMessageReceiptsByStatus(ctx context.Context, messageID, status string) (int64, error)
	CountUnreadMessages(ctx context.Context, roomID, accountID string, lastReadAt *time.Time) (int64, error)
}

//go:generate mockgen -package=projection -destination=query_repos_mock.go -source=query_repos.go
type RoomMemberReadRepository interface {
	ListRoomMembers(ctx context.Context, roomID string) ([]*views.RoomMemberView, error)
	GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*views.RoomMemberView, error)
	SearchMentionCandidates(ctx context.Context, search MentionCandidateSearch) ([]*views.MentionCandidateView, error)
}
