package aggregate

import (
	"errors"
	"go-socket/core/modules/payment/domain/types"
	"go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
	"time"
)

type PaymentTransactionAggregate struct {
	event.AggregateRoot

	PaymentTransactionID         string
	PaymentTransactionType       types.TransactionType
	PaymentTransactionAmount     int64
	PaymentTransactionSenderID   string
	PaymentTransactionReceiverID string
	PaymentTransactionSourceType types.SourceType
	PaymentTransactionCreatedAt  time.Time
	PaymentTransactionUpdatedAt  time.Time
}

func (p *PaymentTransactionAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventPaymentTransactionDeposited{},
		&EventPaymentTransactionWithdrawn{},
		&EventPaymentTransactionTransferred{},
		&EventPaymentTransactionRefunded{},
	)
}

func (p *PaymentTransactionAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *EventPaymentTransactionDeposited:
		return p.onPaymentTransactionDeposited(data)
	case *EventPaymentTransactionWithdrawn:
		return p.onPaymentTransactionWithdrawn(data)
	case *EventPaymentTransactionTransferred:
		return p.onPaymentTransactionTransferred(data)
	case *EventPaymentTransactionRefunded:
		return p.onPaymentTransactionRefunded(data)
	default:
		return stackErr.Error(errors.New("unsupported event type"))
	}
}

func (p *PaymentTransactionAggregate) onPaymentTransactionDeposited(data *EventPaymentTransactionDeposited) error {
	p.PaymentTransactionID = data.PaymentTransactionID
	p.PaymentTransactionAmount = data.PaymentTransactionAmount
	p.PaymentTransactionReceiverID = data.PaymentTransactionReceiverID
	p.PaymentTransactionCreatedAt = data.PaymentTransactionCreatedAt
	p.PaymentTransactionUpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}

func (p *PaymentTransactionAggregate) onPaymentTransactionWithdrawn(data *EventPaymentTransactionWithdrawn) error {
	p.PaymentTransactionID = data.PaymentTransactionID
	p.PaymentTransactionAmount = data.PaymentTransactionAmount
	p.PaymentTransactionCreatedAt = data.PaymentTransactionCreatedAt
	p.PaymentTransactionUpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}

func (p *PaymentTransactionAggregate) onPaymentTransactionTransferred(data *EventPaymentTransactionTransferred) error {
	p.PaymentTransactionID = data.PaymentTransactionID
	p.PaymentTransactionAmount = data.PaymentTransactionAmount
	p.PaymentTransactionReceiverID = data.PaymentTransactionReceiverID
	p.PaymentTransactionCreatedAt = data.PaymentTransactionCreatedAt
	p.PaymentTransactionUpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}

func (p *PaymentTransactionAggregate) onPaymentTransactionRefunded(data *EventPaymentTransactionRefunded) error {
	p.PaymentTransactionID = data.PaymentTransactionID
	p.PaymentTransactionAmount = data.PaymentTransactionAmount
	p.PaymentTransactionCreatedAt = data.PaymentTransactionCreatedAt
	p.PaymentTransactionUpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}
