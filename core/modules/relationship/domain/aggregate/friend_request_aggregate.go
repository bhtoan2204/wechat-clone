package aggregate

import (
	"fmt"
	"time"
	"wechat-clone/core/modules/relationship/domain"
	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"
)

type FriendRequestAggregate struct {
	event.AggregateRoot

	*entity.FriendRequest
}

func NewFriendRequest(friendRequestID string) (*FriendRequestAggregate, error) {
	if friendRequestID == "" {
		return nil, stackErr.Error(domain.ErrEmpty)
	}
	agg := &FriendRequestAggregate{}
	if err := event.InitAggregate(&agg.AggregateRoot, agg, friendRequestID); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (agg *FriendRequestAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventFriendRequestCreated{},
		&EventFriendRequestAccept{},
		&EventFriendRequestReject{},
		&EventFriendRequestCancel{},
	)
}

func (a *FriendRequestAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *EventFriendRequestCreated:
		if data.RequesterID == "" || data.AddresseeID == "" {
			return stackErr.Error(domain.ErrInvalidData)
		}

		a.FriendRequest = &entity.FriendRequest{}
		a.FriendRequest.ID = e.AggregateID
		a.FriendRequest.RequesterID = data.RequesterID
		a.FriendRequest.AddresseeID = data.AddresseeID
		a.FriendRequest.Status = entity.FriendRequestStatusPending
		a.FriendRequest.Message = utils.NullableString(data.Message)
		a.FriendRequest.CreatedAt = data.CreatedAt

		return nil
	case *EventFriendRequestAccept:
		if err := a.Accept(data.AcceptedAt); err != nil {
			return stackErr.Error(err)
		}
		return nil
	case *EventFriendRequestReject:
		if err := a.Reject(data.Reason, data.RejectedAt); err != nil {
			return stackErr.Error(err)
		}
		return nil
	case *EventFriendRequestCancel:
		if err := a.Cancel(data.CancelAt); err != nil {
			return stackErr.Error(err)
		}
		return nil
	default:
		return event.ErrUnsupportedEventType
	}
}

func (a *FriendRequestAggregate) Create(
	requesterID string,
	addresseeID string,
	message *string,
	now time.Time,
) error {
	msg := ""
	if message != nil {
		msg = *message
	}

	return stackErr.Error(a.ApplyChange(a, &EventFriendRequestCreated{
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Message:     msg,
		CreatedAt:   now,
	}))
}

func (a *FriendRequestAggregate) AcceptRequest(now time.Time) error {
	if a.FriendRequest == nil {
		return stackErr.Error(fmt.Errorf("friend request entity is required"))
	}
	return stackErr.Error(a.ApplyChange(a, &EventFriendRequestAccept{
		AcceptedAt: now,
	}))
}

func (a *FriendRequestAggregate) RejectRequest(reason *string, now time.Time) error {
	if a.FriendRequest == nil {
		return stackErr.Error(fmt.Errorf("friend request entity is required"))
	}
	return stackErr.Error(a.ApplyChange(a, &EventFriendRequestReject{
		Reason:     reason,
		RejectedAt: now,
	}))
}

func (a *FriendRequestAggregate) CancelRequest(now time.Time) error {
	if a.FriendRequest == nil {
		return stackErr.Error(fmt.Errorf("friend request entity is required"))
	}
	return stackErr.Error(a.ApplyChange(a, &EventFriendRequestCancel{
		CancelAt: now,
	}))
}

func (a *FriendRequestAggregate) SetFriendRequest(friendRequest *entity.FriendRequest) error {
	if friendRequest == nil {
		return stackErr.Error(domain.ErrEmpty)
	}
	a.FriendRequest = friendRequest
	return nil
}
