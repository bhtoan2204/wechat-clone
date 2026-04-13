package views

import "time"

type RoomMemberView struct {
	ID              string     `db:"id"`
	RoomID          string     `db:"room_id"`
	AccountID       string     `db:"account_id"`
	Role            string     `db:"role"`
	DisplayName     string     `db:"display_name"`
	Username        string     `db:"username"`
	AvatarObjectKey string     `db:"avatar_object_key"`
	LastDeliveredAt *time.Time `db:"last_delivered_at"`
	LastReadAt      *time.Time `db:"last_read_at"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}
