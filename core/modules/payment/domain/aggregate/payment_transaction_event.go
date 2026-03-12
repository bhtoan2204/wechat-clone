package aggregate

import "time"

type EventPaymentTransactionDeposited struct {
	PaymentTransactionID         string
	PaymentTransactionAmount     int64
	PaymentTransactionReceiverID string
	PaymentTransactionCreatedAt  time.Time
	PaymentTransactionUpdatedAt  time.Time
}

type EventPaymentTransactionWithdrawn struct {
	PaymentTransactionID        string
	PaymentTransactionAmount    int64
	PaymentTransactionCreatedAt time.Time
	PaymentTransactionUpdatedAt time.Time
}

type EventPaymentTransactionTransferred struct {
	PaymentTransactionID         string
	PaymentTransactionAmount     int64
	PaymentTransactionReceiverID string
	PaymentTransactionCreatedAt  time.Time
	PaymentTransactionUpdatedAt  time.Time
}

type EventPaymentTransactionRefunded struct {
	PaymentTransactionID        string
	PaymentTransactionAmount    int64
	PaymentTransactionCreatedAt time.Time
	PaymentTransactionUpdatedAt time.Time
}
