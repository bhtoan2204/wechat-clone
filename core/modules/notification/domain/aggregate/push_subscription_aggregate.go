package aggregate

import (
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/notification/domain/entity"
	"go-socket/core/shared/pkg/stackErr"
)

var (
	ErrPushSubscriptionAggregateNotInitialized = errors.New("push subscription aggregate is not initialized")
	ErrPushSubscriptionIDRequired              = errors.New("push subscription id is required")
	ErrPushSubscriptionAccountIDRequired       = errors.New("push subscription account_id is required")
	ErrPushSubscriptionEndpointRequired        = errors.New("push subscription endpoint is required")
	ErrPushSubscriptionKeysRequired            = errors.New("push subscription keys are required")
	ErrPushSubscriptionOccurredAtRequired      = errors.New("push subscription occurred_at is required")
)

type PushSubscriptionAggregate struct {
	subscription *entity.PushSubscription
}

func NewPushSubscriptionAggregate(id string) (*PushSubscriptionAggregate, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, stackErr.Error(ErrPushSubscriptionIDRequired)
	}

	return &PushSubscriptionAggregate{
		subscription: &entity.PushSubscription{ID: id},
	}, nil
}

func (a *PushSubscriptionAggregate) Create(accountID, endpoint, keys string, now time.Time) error {
	if a == nil || a.subscription == nil || strings.TrimSpace(a.subscription.ID) == "" {
		return stackErr.Error(ErrPushSubscriptionAggregateNotInitialized)
	}

	accountID = strings.TrimSpace(accountID)
	endpoint = strings.TrimSpace(endpoint)
	keys = strings.TrimSpace(keys)
	now, err := normalizePushSubscriptionTime(now)
	if err != nil {
		return stackErr.Error(err)
	}

	switch {
	case accountID == "":
		return stackErr.Error(ErrPushSubscriptionAccountIDRequired)
	case endpoint == "":
		return stackErr.Error(ErrPushSubscriptionEndpointRequired)
	case keys == "":
		return stackErr.Error(ErrPushSubscriptionKeysRequired)
	}

	a.subscription.AccountID = accountID
	a.subscription.Endpoint = endpoint
	a.subscription.Keys = keys
	a.subscription.CreatedAt = now
	a.subscription.UpdatedAt = now
	return nil
}

func (a *PushSubscriptionAggregate) UpdateKeys(keys string, now time.Time) (bool, error) {
	if a == nil || a.subscription == nil {
		return false, stackErr.Error(ErrPushSubscriptionAggregateNotInitialized)
	}

	keys = strings.TrimSpace(keys)
	now, err := normalizePushSubscriptionTime(now)
	if err != nil {
		return false, stackErr.Error(err)
	}

	if keys == "" {
		return false, stackErr.Error(ErrPushSubscriptionKeysRequired)
	}
	if a.subscription.Keys == keys {
		return false, nil
	}

	a.subscription.Keys = keys
	a.subscription.UpdatedAt = now
	return true, nil
}

func (a *PushSubscriptionAggregate) Restore(subscription *entity.PushSubscription) error {
	if subscription == nil {
		return stackErr.Error(ErrPushSubscriptionAggregateNotInitialized)
	}

	a.subscription = &entity.PushSubscription{
		ID:        strings.TrimSpace(subscription.ID),
		AccountID: strings.TrimSpace(subscription.AccountID),
		Endpoint:  strings.TrimSpace(subscription.Endpoint),
		Keys:      strings.TrimSpace(subscription.Keys),
		CreatedAt: subscription.CreatedAt.UTC(),
		UpdatedAt: subscription.UpdatedAt.UTC(),
	}
	return nil
}

func (a *PushSubscriptionAggregate) Snapshot() (*entity.PushSubscription, error) {
	if a == nil || a.subscription == nil {
		return nil, stackErr.Error(ErrPushSubscriptionAggregateNotInitialized)
	}

	switch {
	case strings.TrimSpace(a.subscription.ID) == "":
		return nil, stackErr.Error(ErrPushSubscriptionIDRequired)
	case strings.TrimSpace(a.subscription.AccountID) == "":
		return nil, stackErr.Error(ErrPushSubscriptionAccountIDRequired)
	case strings.TrimSpace(a.subscription.Endpoint) == "":
		return nil, stackErr.Error(ErrPushSubscriptionEndpointRequired)
	case strings.TrimSpace(a.subscription.Keys) == "":
		return nil, stackErr.Error(ErrPushSubscriptionKeysRequired)
	}

	clone := *a.subscription
	clone.CreatedAt = a.subscription.CreatedAt.UTC()
	clone.UpdatedAt = a.subscription.UpdatedAt.UTC()
	return &clone, nil
}

func normalizePushSubscriptionTime(value time.Time) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, ErrPushSubscriptionOccurredAtRequired
	}
	return value.UTC(), nil
}
