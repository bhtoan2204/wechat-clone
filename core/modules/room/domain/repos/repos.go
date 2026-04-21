package repos

import (
	"context"
)

//go:generate mockgen -package=repos -destination=repos_mock.go -source=repos.go
type Repos interface {
	RoomAggregateRepository() RoomAggregateRepository
	MessageAggregateRepository() MessageAggregateRepository

	RoomRepository() RoomRepository
	MessageRepository() MessageRepository
	RoomMemberRepository() RoomMemberRepository
	RoomOutboxEventsRepository() RoomOutboxEventsRepository

	RoomAccountRepository() RoomAccountRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
