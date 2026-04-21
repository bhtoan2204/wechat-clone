package models

import "time"

type RelationshipAccount struct {
	AccountID       string    `gorm:"column:account_id;type:varchar(36);primaryKey"`
	DisplayName     string    `gorm:"column:display_name;type:varchar(255);not null;default:''"`
	Username        string    `gorm:"column:username;type:varchar(255);not null;default:''"`
	AvatarObjectKey string    `gorm:"column:avatar_object_key;type:varchar(2048);not null;default:''"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;not null;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;not null;autoUpdateTime"`
}

func (RelationshipAccount) TableName() string {
	return "relationship_accounts"
}
