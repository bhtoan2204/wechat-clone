package models

import "time"

// DeviceModel stores a known device owned by a specific account.
type DeviceModel struct {
	ID            string `gorm:"primaryKey"`
	AccountID     string `gorm:"not null"`
	DeviceUID     string `gorm:"not null"` // stable ID of client/app
	DeviceName    *string
	DeviceType    string `gorm:"not null;default:web"` // web, ios, android, desktop, other
	OSName        *string
	OSVersion     *string
	AppVersion    *string
	UserAgent     *string
	LastIPAddress *string
	LastSeenAt    *time.Time `gorm:"index:ix_dev_seen"`
	IsTrusted     int8       `gorm:"not null;default:0"` // Oracle-friendly: 0/1
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (DeviceModel) TableName() string {
	return "devices"
}
