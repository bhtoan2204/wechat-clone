package entity

import "time"

type PaymentHistory struct {
	ID           string
	Type         string
	Amount       int64
	Balance      int64
	SenderID     string
	ReceiverID   string
	Sender       PaymentAccount
	Receiver     PaymentAccount
	SenderName   string
	ReceiverName string
	CreatedAt    time.Time
}
