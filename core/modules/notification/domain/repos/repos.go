package repos

import "context"

//go:generate mockgen -package=repos -destination=repos_mock.go -source=repos.go
type Repos interface {
	NotificationRepository() NotificationRepository
	PushSubscriptionRepository() PushSubscriptionRepository

	WithTransaction(ctx context.Context, fn func(Repos) error) error
}
