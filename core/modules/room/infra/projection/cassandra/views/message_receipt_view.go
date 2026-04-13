package views

import "time"

type MessageReceiptView struct {
	ID          string     `db:"id"`
	MessageID   string     `db:"message_id"`
	AccountID   string     `db:"account_id"`
	Status      string     `db:"status"`
	DeliveredAt *time.Time `db:"delivered_at"`
	SeenAt      *time.Time `db:"seen_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}
