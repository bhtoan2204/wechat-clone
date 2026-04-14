package repos

import "errors"

var (
	ErrNotificationNotFound     = errors.New("notification not found")
	ErrPushSubscriptionNotFound = errors.New("push subscription not found")
)
