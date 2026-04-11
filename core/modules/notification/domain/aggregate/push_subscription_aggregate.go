package aggregate

import (
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/notification/domain/entity"
)

var (
	ErrPushSubscriptionIDRequired        = errors.New("push subscription id is required")
	ErrPushSubscriptionAccountIDRequired = errors.New("push subscription account_id is required")
	ErrPushSubscriptionEndpointRequired  = errors.New("push subscription endpoint is required")
	ErrPushSubscriptionKeysRequired      = errors.New("push subscription keys are required")
)

type PushSubscriptionAggregate struct {
	ID        string
	AccountID string
	Endpoint  string
	Keys      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewPushSubscriptionAggregate(id string) (*PushSubscriptionAggregate, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrPushSubscriptionIDRequired
	}

	return &PushSubscriptionAggregate{ID: id}, nil
}

func (a *PushSubscriptionAggregate) Create(accountID, endpoint, keys string, now time.Time) error {
	accountID = strings.TrimSpace(accountID)
	endpoint = strings.TrimSpace(endpoint)
	keys = strings.TrimSpace(keys)

	switch {
	case accountID == "":
		return ErrPushSubscriptionAccountIDRequired
	case endpoint == "":
		return ErrPushSubscriptionEndpointRequired
	case keys == "":
		return ErrPushSubscriptionKeysRequired
	}

	normalizedNow := now.UTC()
	a.AccountID = accountID
	a.Endpoint = endpoint
	a.Keys = keys
	a.CreatedAt = normalizedNow
	a.UpdatedAt = normalizedNow
	return nil
}

func (a *PushSubscriptionAggregate) UpdateKeys(keys string, now time.Time) (bool, error) {
	keys = strings.TrimSpace(keys)
	if keys == "" {
		return false, ErrPushSubscriptionKeysRequired
	}
	if a.Keys == keys {
		return false, nil
	}

	a.Keys = keys
	a.UpdatedAt = now.UTC()
	return true, nil
}

func (a *PushSubscriptionAggregate) Restore(subscription *entity.PushSubscription) error {
	if subscription == nil {
		return nil
	}

	a.ID = strings.TrimSpace(subscription.ID)
	a.AccountID = strings.TrimSpace(subscription.AccountID)
	a.Endpoint = strings.TrimSpace(subscription.Endpoint)
	a.Keys = strings.TrimSpace(subscription.Keys)
	a.CreatedAt = subscription.CreatedAt.UTC()
	a.UpdatedAt = subscription.UpdatedAt.UTC()
	return nil
}

func (a *PushSubscriptionAggregate) Snapshot() (*entity.PushSubscription, error) {
	switch {
	case strings.TrimSpace(a.ID) == "":
		return nil, ErrPushSubscriptionIDRequired
	case strings.TrimSpace(a.AccountID) == "":
		return nil, ErrPushSubscriptionAccountIDRequired
	case strings.TrimSpace(a.Endpoint) == "":
		return nil, ErrPushSubscriptionEndpointRequired
	case strings.TrimSpace(a.Keys) == "":
		return nil, ErrPushSubscriptionKeysRequired
	}

	return &entity.PushSubscription{
		ID:        a.ID,
		AccountID: a.AccountID,
		Endpoint:  a.Endpoint,
		Keys:      a.Keys,
		CreatedAt: a.CreatedAt.UTC(),
		UpdatedAt: a.UpdatedAt.UTC(),
	}, nil
}
