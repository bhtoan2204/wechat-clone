package models

import "time"

// SessionModel stores one login or refresh-token session bound to a known device.
type SessionModel struct {
	ID               string `gorm:"primaryKey"`
	AccountID        string `gorm:"not null"`
	DeviceID         string `gorm:"not null"`
	RefreshTokenHash string `gorm:"not null"`
	Status           string `gorm:"not null;default:active"` // active, revoked, expired
	IPAddress        *string
	UserAgent        *string
	LastActivityAt   *time.Time `gorm:"index:ix_ses_last_act"`
	ExpiresAt        time.Time  `gorm:"not null;index:ix_ses_exp"`
	RevokedAt        *time.Time
	RevokedReason    *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (SessionModel) TableName() string {
	return "sessions"
}
