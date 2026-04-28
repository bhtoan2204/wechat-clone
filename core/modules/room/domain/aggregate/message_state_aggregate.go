package aggregate

import (
	"errors"
	"strings"
	"time"

	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/shared/pkg/stackErr"
)

var ErrMessageAggregateNil = errors.New("message aggregate is nil")

type PendingMessageDeletion struct {
	MessageID string
	AccountID string
	CreatedAt time.Time
}

type MessageStateAggregate struct {
	message         *entity.MessageEntity
	recipientMember *entity.RoomMemberEntity
	messageDirty    bool
	memberDirty     bool
	pendingDeletion *PendingMessageDeletion
	pendingReceipt  *PendingMessageReceipt
}

func NewMessageStateAggregate(message *entity.MessageEntity) (*MessageStateAggregate, error) {
	if message == nil {
		return nil, stackErr.Error(ErrMessageAggregateNil)
	}
	return &MessageStateAggregate{message: message}, nil
}

func NewMessageStateAggregateForRecipient(message *entity.MessageEntity, recipientMember *entity.RoomMemberEntity) (*MessageStateAggregate, error) {
	if message == nil {
		return nil, stackErr.Error(ErrMessageAggregateNil)
	}
	return &MessageStateAggregate{message: message, recipientMember: recipientMember}, nil
}

func (a *MessageStateAggregate) Message() *entity.MessageEntity {
	return a.message
}

func (a *MessageStateAggregate) RecipientMember() *entity.RoomMemberEntity {
	return a.recipientMember
}

func (a *MessageStateAggregate) PendingReceipt() *PendingMessageReceipt {
	return a.pendingReceipt
}

func (a *MessageStateAggregate) PendingDeletion() *PendingMessageDeletion {
	return a.pendingDeletion
}

func (a *MessageStateAggregate) MessageDirty() bool {
	return a.messageDirty
}

func (a *MessageStateAggregate) MemberDirty() bool {
	return a.memberDirty
}

func (a *MessageStateAggregate) MarkPersisted() {
	a.messageDirty = false
	a.memberDirty = false
	a.pendingDeletion = nil
	a.pendingReceipt = nil
}

func (a *MessageStateAggregate) Edit(actorID, content string, editedAt time.Time) error {
	if a == nil || a.message == nil {
		return stackErr.Error(ErrMessageAggregateNil)
	}
	if err := a.message.Edit(actorID, content, editedAt); err != nil {
		return stackErr.Error(err)
	}
	a.messageDirty = true
	return nil
}

func (a *MessageStateAggregate) ToggleReaction(accountID, emoji string, reactedAt time.Time) error {
	if a == nil || a.message == nil {
		return stackErr.Error(ErrMessageAggregateNil)
	}
	changed, err := a.message.ToggleReaction(accountID, emoji, reactedAt)
	if err != nil {
		return stackErr.Error(err)
	}
	if changed {
		a.messageDirty = true
	}
	return nil
}

func (a *MessageStateAggregate) Delete(actorID, accountID, scope string, now time.Time) error {
	if a == nil || a.message == nil {
		return stackErr.Error(ErrMessageAggregateNil)
	}

	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "me":
		a.pendingDeletion = &PendingMessageDeletion{
			MessageID: a.message.ID,
			AccountID: strings.TrimSpace(accountID),
			CreatedAt: now.UTC(),
		}
		return nil
	case "everyone":
		if err := a.message.DeleteForEveryone(actorID, now); err != nil {
			return stackErr.Error(err)
		}
		a.messageDirty = true
		return nil
	default:
		return stackErr.Error(errors.New("scope must be one of: me, everyone"))
	}
}

func (a *MessageStateAggregate) MarkStatus(accountID, status string, member *entity.RoomMemberEntity, now time.Time) (bool, error) {
	if a == nil || a.message == nil {
		return false, stackErr.Error(ErrMessageAggregateNil)
	}

	accountID = strings.TrimSpace(accountID)
	if accountID == "" || accountID == strings.TrimSpace(a.message.SenderID) {
		return false, nil
	}

	normalizedStatus, err := entity.NormalizeReceiptStatus(status)
	if err != nil {
		return false, stackErr.Error(err)
	}

	deliveredValue := now.UTC()
	deliveredAt := &deliveredValue
	var seenAt *time.Time

	if member == nil {
		member = a.recipientMember
	}
	a.recipientMember = member
	if member != nil {
		appliedStatus, appliedDeliveredAt, appliedSeenAt, applyErr := member.ApplyReceiptStatus(normalizedStatus, now)
		if applyErr != nil {
			return false, stackErr.Error(applyErr)
		}
		normalizedStatus = appliedStatus
		deliveredAt = appliedDeliveredAt
		seenAt = appliedSeenAt
		a.memberDirty = true
	} else if normalizedStatus == "seen" {
		seenAt = deliveredAt
	}

	a.pendingReceipt = &PendingMessageReceipt{
		MessageID:   a.message.ID,
		AccountID:   strings.TrimSpace(accountID),
		Status:      normalizedStatus,
		DeliveredAt: deliveredAt,
		SeenAt:      seenAt,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
	return true, nil
}
