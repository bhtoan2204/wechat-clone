package views

import "time"

type MessageDeletionView struct {
	ID        string    `db:"id"`
	MessageID string    `db:"message_id"`
	AccountID string    `db:"account_id"`
	CreatedAt time.Time `db:"created_at"`
}
