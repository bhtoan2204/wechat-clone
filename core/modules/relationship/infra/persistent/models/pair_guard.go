package models

import "time"

type RelationshipPairGuard struct {
	UserLowID  string    `gorm:"column:user_low_id;type:varchar(36);primaryKey"`
	UserHighID string    `gorm:"column:user_high_id;type:varchar(36);primaryKey"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;not null;autoCreateTime"`
}

func (RelationshipPairGuard) TableName() string {
	return "relationship_pair_guards"
}
