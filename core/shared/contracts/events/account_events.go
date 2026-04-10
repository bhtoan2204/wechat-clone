package events

import "time"

const (
	EventAccountCreated        = "EventAccountCreated"
	EventAccountUpdated        = "EventAccountUpdated"
	EventAccountProfileUpdated = "EventAccountProfileUpdated"
	EventAccountBanned         = "EventAccountBanned"
)

type AccountCreatedEvent struct {
	AccountID   string
	Email       string
	DisplayName string
	CreatedAt   time.Time
}

type AccountUpdatedEvent struct {
	AccountID string
	Email     string
	UpdatedAt time.Time
}

type AccountProfileUpdatedEvent struct {
	AccountID       string
	DisplayName     string
	Username        *string
	AvatarObjectKey *string
	UpdatedAt       time.Time
}

type AccountBannedEvent struct {
	AccountID string
	BanReason string
	BanUntil  *time.Time
}

func (e *AccountCreatedEvent) GetName() string {
	return EventAccountCreated
}

func (e *AccountCreatedEvent) GetData() interface{} {
	return e
}

func (e *AccountUpdatedEvent) GetName() string {
	return EventAccountUpdated
}

func (e *AccountUpdatedEvent) GetData() interface{} {
	return e
}

func (e *AccountProfileUpdatedEvent) GetName() string {
	return EventAccountProfileUpdated
}

func (e *AccountProfileUpdatedEvent) GetData() interface{} {
	return e
}

func (e *AccountBannedEvent) GetName() string {
	return EventAccountBanned
}

func (e *AccountBannedEvent) GetData() interface{} {
	return e
}
