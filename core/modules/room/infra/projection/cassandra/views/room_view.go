package views

import "time"

type RoomView struct {
	ID                  string     `db:"id"`
	Name                string     `db:"name"`
	Description         string     `db:"description"`
	RoomType            string     `db:"room_type"`
	OwnerID             string     `db:"owner_id"`
	DirectKey           *string    `db:"direct_key"`
	PinnedMessageID     *string    `db:"pinned_message_id"`
	MemberCount         int        `db:"member_count"`
	LastMessageID       *string    `db:"last_message_id"`
	LastMessageAt       *time.Time `db:"last_message_at"`
	LastMessageContent  *string    `db:"last_message_content"`
	LastMessageSenderID *string    `db:"last_message_sender_id"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}
