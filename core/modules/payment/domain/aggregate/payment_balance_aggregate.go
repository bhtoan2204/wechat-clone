package aggregate

import (
	"errors"
	"reflect"
	"time"

	"go-socket/core/shared/pkg/event"
	stackerr "go-socket/core/shared/pkg/stackErr"
)

var (
	ErrInvalidPaymentAmount = errors.New("amount must be greater than 0")
	ErrInsufficientBalance  = errors.New("insufficient balance")
)

type PaymentBalanceAggregate struct {
	event.AggregateRoot

	AccountID string
	Balance   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewPaymentBalanceAggregate(accountID string) (*PaymentBalanceAggregate, error) {
	agg := &PaymentBalanceAggregate{}
	agg.SetAggregateType(reflect.TypeOf(agg).Elem().Name())
	if err := agg.SetID(accountID); err != nil {
		return nil, stackerr.Error(err)
	}

	return agg, nil
}

func (p *PaymentBalanceAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventPaymentTransactionDeposited{},
		&EventPaymentTransactionWithdrawn{},
		&EventPaymentTransactionTransferred{},
		&EventPaymentTransactionReceived{},
	)
}

func (p *PaymentBalanceAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *EventPaymentTransactionDeposited:
		return p.onDeposited(e.AggregateID, data)
	case *EventPaymentTransactionWithdrawn:
		return p.onWithdrawn(e.AggregateID, data)
	case *EventPaymentTransactionTransferred:
		return p.onTransferred(e.AggregateID, data)
	case *EventPaymentTransactionReceived:
		return p.onReceived(e.AggregateID, data)
	default:
		return errors.New("unsupported event type")
	}
}

func (p *PaymentBalanceAggregate) Deposit(transactionID string, amount int64, now time.Time) error {
	if amount <= 0 {
		return ErrInvalidPaymentAmount
	}

	return p.ApplyChange(p, &EventPaymentTransactionDeposited{
		PaymentTransactionID:         transactionID,
		PaymentTransactionAmount:     amount,
		PaymentTransactionReceiverID: p.AccountID,
		PaymentTransactionCreatedAt:  now,
		PaymentTransactionUpdatedAt:  now,
	})
}

func (p *PaymentBalanceAggregate) Withdraw(transactionID string, amount int64, now time.Time) error {
	if amount <= 0 {
		return ErrInvalidPaymentAmount
	}
	if p.Balance < amount {
		return ErrInsufficientBalance
	}

	return p.ApplyChange(p, &EventPaymentTransactionWithdrawn{
		PaymentTransactionID:        transactionID,
		PaymentTransactionAmount:    amount,
		PaymentTransactionCreatedAt: now,
		PaymentTransactionUpdatedAt: now,
	})
}

func (p *PaymentBalanceAggregate) Transfer(transactionID string, amount int64, receiverID string, now time.Time) error {
	if amount <= 0 {
		return ErrInvalidPaymentAmount
	}
	if p.Balance < amount {
		return ErrInsufficientBalance
	}

	return p.ApplyChange(p, &EventPaymentTransactionTransferred{
		PaymentTransactionID:         transactionID,
		PaymentTransactionAmount:     amount,
		PaymentTransactionReceiverID: receiverID,
		PaymentTransactionCreatedAt:  now,
		PaymentTransactionUpdatedAt:  now,
	})
}

func (p *PaymentBalanceAggregate) Receive(transactionID string, amount int64, senderID string, now time.Time) error {
	if amount <= 0 {
		return ErrInvalidPaymentAmount
	}

	return p.ApplyChange(p, &EventPaymentTransactionReceived{
		PaymentTransactionID:         transactionID,
		PaymentTransactionAmount:     amount,
		PaymentTransactionSenderID:   senderID,
		PaymentTransactionReceiverID: p.AccountID,
		PaymentTransactionCreatedAt:  now,
		PaymentTransactionUpdatedAt:  now,
	})
}

func (p *PaymentBalanceAggregate) onDeposited(accountID string, data *EventPaymentTransactionDeposited) error {
	p.AccountID = accountID
	p.Balance += data.PaymentTransactionAmount
	if p.CreatedAt.IsZero() {
		p.CreatedAt = data.PaymentTransactionCreatedAt
	}
	p.UpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}

func (p *PaymentBalanceAggregate) onWithdrawn(accountID string, data *EventPaymentTransactionWithdrawn) error {
	p.AccountID = accountID
	p.Balance -= data.PaymentTransactionAmount
	if p.CreatedAt.IsZero() {
		p.CreatedAt = data.PaymentTransactionCreatedAt
	}
	p.UpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}

func (p *PaymentBalanceAggregate) onTransferred(accountID string, data *EventPaymentTransactionTransferred) error {
	p.AccountID = accountID
	p.Balance -= data.PaymentTransactionAmount
	if p.CreatedAt.IsZero() {
		p.CreatedAt = data.PaymentTransactionCreatedAt
	}
	p.UpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}

func (p *PaymentBalanceAggregate) onReceived(accountID string, data *EventPaymentTransactionReceived) error {
	p.AccountID = accountID
	p.Balance += data.PaymentTransactionAmount
	if p.CreatedAt.IsZero() {
		p.CreatedAt = data.PaymentTransactionCreatedAt
	}
	p.UpdatedAt = data.PaymentTransactionUpdatedAt
	return nil
}
