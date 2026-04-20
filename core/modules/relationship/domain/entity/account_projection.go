package entity

import "time"

type AccountProjection struct {
	AccountID       string
	DisplayName     string
	Username        string
	AvatarObjectKey string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
