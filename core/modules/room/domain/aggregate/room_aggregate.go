package aggregate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"wechat-clone/core/modules/room/domain/entity"
	roomtypes "wechat-clone/core/modules/room/types"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

var (
	ErrRoomAggregateIDMismatch     = errors.New("room id mismatch")
	ErrRoomEventOccurredAtRequired = errors.New("room event occurred_at is required")
	ErrRoomAggregateNil            = errors.New("room aggregate is nil")
)

type PendingMessageReceipt struct {
	MessageID   string
	AccountID   string
	Status      string
	DeliveredAt *time.Time
	SeenAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PendingRoomOutboxEvent struct {
	EventName string
	Payload   interface{}
	CreatedAt time.Time
}

type MessageSenderIdentity struct {
	Name  string
	Email string
}

type MessageOutboxPayload struct {
	Mentions            []sharedevents.RoomMessageMention
	MentionAll          bool
	MentionedAccountIDs []string
}

type UpdateGroupDetailsParams struct {
	ActorID       string
	Name          string
	Description   string
	Now           time.Time
	SystemActorID string
}

func NewConversationRoomAggregate(
	room *entity.Room,
	members []*entity.RoomMemberEntity,
	systemActorID string,
	systemMessage string,
	now time.Time,
) (*RoomAggregate, error) {
	agg, err := RestoreRoomAggregate(room, members, 0)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	agg.isNew = true
	agg.roomDirty = true
	for _, member := range agg.Members() {
		if member == nil {
			continue
		}
		agg.memberUpserts[strings.TrimSpace(member.AccountID)] = member
	}
	if err := agg.RecordCreated(len(agg.Members()), room.CreatedAt); err != nil {
		return nil, stackErr.Error(err)
	}
	if strings.TrimSpace(systemMessage) != "" {
		if _, err := agg.appendSystemMessage(systemActorID, systemMessage, now); err != nil {
			return nil, stackErr.Error(err)
		}
	}
	return agg, nil
}

func RestoreRoomAggregate(room *entity.Room, members []*entity.RoomMemberEntity, baseVersion int) (*RoomAggregate, error) {
	if room == nil {
		return nil, stackErr.Error(ErrRoomAggregateNil)
	}

	agg := &RoomAggregate{
		room:          room,
		members:       make(map[string]*entity.RoomMemberEntity, len(members)),
		memberOrder:   make([]string, 0, len(members)),
		memberUpserts: make(map[string]*entity.RoomMemberEntity),
	}
	if err := event.InitAggregate(&agg.AggregateRoot, agg, room.ID); err != nil {
		return nil, stackErr.Error(err)
	}
	agg.SetInternal(room.ID, baseVersion, baseVersion)
	for _, member := range members {
		if err := agg.attachRestoredMember(member); err != nil {
			return nil, stackErr.Error(err)
		}
	}
	return agg, nil
}

type RoomAggregate struct {
	event.AggregateRoot

	room             *entity.Room
	members          map[string]*entity.RoomMemberEntity
	memberOrder      []string
	isNew            bool
	roomDirty        bool
	roomDeleted      bool
	pendingMessages  []*entity.MessageEntity
	pendingReceipts  []PendingMessageReceipt
	memberUpserts    map[string]*entity.RoomMemberEntity
	removedMemberIDs []string
}

func (r *RoomAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventRoomCreated{},
		&EventRoomOwnerChanged{},
		&EventRoomDetailsUpdated{},
		&EventRoomMessagePinned{},
		&EventRoomMemberAdded{},
		&EventRoomMemberRemoved{},
		&EventRoomMessageCreated{},
		&sharedevents.RoomMessageCreatedEvent{},
	)
}

func (r *RoomAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *EventRoomCreated:
		return r.applyRoomCreated(e.AggregateID, data)
	case *EventRoomOwnerChanged:
		return r.ensureRoomID(data.RoomID)
	case *EventRoomDetailsUpdated:
		return r.ensureRoomID(data.RoomID)
	case *EventRoomMessagePinned:
		return r.ensureRoomID(data.RoomID)
	case *EventRoomMemberAdded:
		return r.applyRoomMemberAdded(data)
	case *EventRoomMemberRemoved:
		return r.applyRoomMemberRemoved(data)
	case *EventRoomMessageCreated:
		return r.applyRoomMessageCreated(data.RoomID, data.MessageID, data.MessageContent, data.MessageSentAt)
	case *sharedevents.RoomMessageCreatedEvent:
		return r.applyRoomMessageCreated(data.RoomID, data.MessageID, data.MessageContent, data.MessageSentAt)
	default:
		return event.ErrUnsupportedEventType
	}
}

func (r *RoomAggregate) applyRoomCreated(
	aggregateID string,
	data *EventRoomCreated,
) error {
	if r.room == nil {
		r.room = &entity.Room{ID: aggregateID}
	}
	r.room.ID = aggregateID
	r.room.RoomType = data.RoomType
	return nil
}

func (r *RoomAggregate) applyRoomMemberAdded(
	data *EventRoomMemberAdded,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return err
	}

	return nil
}

func (r *RoomAggregate) applyRoomMemberRemoved(
	data *EventRoomMemberRemoved,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return err
	}

	return nil
}

func (r *RoomAggregate) applyRoomMessageCreated(
	roomID string,
	messageID string,
	messageContent string,
	messageSentAt time.Time,
) error {
	if err := r.ensureRoomID(roomID); err != nil {
		return err
	}

	_ = messageID
	_ = messageContent
	_ = messageSentAt

	return nil
}

func (r *RoomAggregate) ensureRoomID(roomID string) error {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return entity.ErrRoomIDRequired
	}
	if r.AggregateID() == "" {
		return r.SetID(roomID)
	}
	if r.AggregateID() != roomID {
		return ErrRoomAggregateIDMismatch
	}
	return nil
}

func (a *RoomAggregate) RecordCreated(memberCount int, now time.Time) error {
	if a == nil || a.room == nil {
		return stackErr.Error(ErrRoomAggregateNil)
	}
	return stackErr.Error(a.recordEvent(&EventRoomCreated{
		RoomID:      a.room.ID,
		RoomType:    a.room.RoomType,
		MemberCount: memberCount,
	}, now))
}

func (a *RoomAggregate) Room() *entity.Room {
	return a.room
}

func (a *RoomAggregate) BaseVersion() int {
	return a.AggregateRoot.BaseVersion()
}

func (a *RoomAggregate) IsNew() bool {
	return a.isNew
}

func (a *RoomAggregate) IsDeleted() bool {
	return a.roomDeleted
}

func (a *RoomAggregate) Members() []*entity.RoomMemberEntity {
	results := make([]*entity.RoomMemberEntity, 0, len(a.memberOrder))
	for _, accountID := range a.memberOrder {
		member, ok := a.members[accountID]
		if !ok || member == nil {
			continue
		}
		results = append(results, member)
	}
	return results
}

func (a *RoomAggregate) PendingMessages() []*entity.MessageEntity {
	return append([]*entity.MessageEntity(nil), a.pendingMessages...)
}

func (a *RoomAggregate) PendingReceipts() []PendingMessageReceipt {
	return append([]PendingMessageReceipt(nil), a.pendingReceipts...)
}

func (a *RoomAggregate) PendingOutboxEvents() []PendingRoomOutboxEvent {
	events := a.CloneEvents()
	results := make([]PendingRoomOutboxEvent, 0, len(events))
	for _, evt := range events {
		createdAt := time.Now().UTC()
		if evt.CreatedAt > 0 {
			createdAt = time.UnixMilli(evt.CreatedAt).UTC()
		}
		results = append(results, PendingRoomOutboxEvent{
			EventName: evt.EventName,
			Payload:   evt.EventData,
			CreatedAt: createdAt,
		})
	}
	return results
}

func (a *RoomAggregate) PendingMemberUpserts() []*entity.RoomMemberEntity {
	results := make([]*entity.RoomMemberEntity, 0, len(a.memberUpserts))
	for _, member := range a.memberUpserts {
		if member == nil {
			continue
		}
		results = append(results, member)
	}
	return results
}

func (a *RoomAggregate) RemovedMemberIDs() []string {
	return append([]string(nil), a.removedMemberIDs...)
}

func (a *RoomAggregate) MarkPersisted(baseVersion int) {
	a.SetInternal(a.room.ID, baseVersion, baseVersion)
	a.AggregateRoot.MarkPersisted()
	a.isNew = false
	a.roomDirty = false
	a.pendingMessages = nil
	a.pendingReceipts = nil
	a.memberUpserts = make(map[string]*entity.RoomMemberEntity)
	a.removedMemberIDs = nil
}

func (a *RoomAggregate) ChangeOwner(ownerID string, updatedAt time.Time) (bool, error) {
	if a == nil || a.room == nil {
		return false, stackErr.Error(ErrRoomAggregateNil)
	}
	previousOwnerID := a.room.OwnerID
	changed, err := a.room.ChangeOwner(ownerID, updatedAt)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if !changed {
		return false, nil
	}

	a.roomDirty = true
	if err := a.recordEvent(&EventRoomOwnerChanged{
		RoomID:          a.room.ID,
		PreviousOwnerID: previousOwnerID,
		OwnerID:         a.room.OwnerID,
		ChangedAt:       a.room.UpdatedAt,
	}, a.room.UpdatedAt); err != nil {
		return false, stackErr.Error(err)
	}
	return true, nil
}

func (a *RoomAggregate) UpdateRoomDetails(name, description string, roomType roomtypes.RoomType, updatedAt time.Time) (bool, error) {
	if a == nil || a.room == nil {
		return false, stackErr.Error(ErrRoomAggregateNil)
	}

	updated, err := a.room.UpdateDetails(name, description, roomType, updatedAt)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if !updated {
		return false, nil
	}

	a.roomDirty = true
	if err := a.recordRoomDetailsUpdated(a.room.UpdatedAt); err != nil {
		return false, stackErr.Error(err)
	}
	return true, nil
}

func (a *RoomAggregate) UpdateGroupDetails(params UpdateGroupDetailsParams) (bool, error) {
	actor, err := a.requireMember(params.ActorID)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if err := actor.CanManageGroup(a.room); err != nil {
		return false, stackErr.Error(err)
	}

	updated, err := a.room.UpdateDetails(params.Name, params.Description, "", params.Now)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if !updated {
		return false, nil
	}

	a.roomDirty = true
	if err := a.recordRoomDetailsUpdated(a.room.UpdatedAt); err != nil {
		return false, stackErr.Error(err)
	}
	if _, err := a.appendSystemMessage(params.SystemActorID, fmt.Sprintf("group renamed to %s", a.room.Name), params.Now); err != nil {
		return false, stackErr.Error(err)
	}
	return true, nil
}

func (a *RoomAggregate) AddMember(actorID string, member *entity.RoomMemberEntity, now time.Time, systemActorID string) (bool, error) {
	actor, err := a.requireMember(actorID)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if err := actor.CanManageGroup(a.room); err != nil {
		return false, stackErr.Error(err)
	}
	if member == nil {
		return false, stackErr.Error(entity.ErrRoomMemberRequired)
	}
	if _, exists := a.members[strings.TrimSpace(member.AccountID)]; exists {
		return false, nil
	}

	if err := a.attachMember(member); err != nil {
		return false, stackErr.Error(err)
	}
	a.room.Touch(now)
	a.roomDirty = true
	if err := a.recordEvent(&EventRoomMemberAdded{
		RoomID:               a.room.ID,
		MemberID:             member.AccountID,
		RoomMemberID:         member.ID,
		MemberName:           member.DisplayName,
		MemberUsername:       member.Username,
		MemberAvatarKey:      member.AvatarObjectKey,
		MemberRole:           member.Role,
		MemberJoinedAt:       member.CreatedAt,
		MemberStateUpdatedAt: member.UpdatedAt,
	}, now); err != nil {
		return false, stackErr.Error(err)
	}
	if _, err := a.appendSystemMessage(systemActorID, fmt.Sprintf("%s joined", member.AccountID), now); err != nil {
		return false, stackErr.Error(err)
	}
	return true, nil
}

func (a *RoomAggregate) RemoveMember(actorID, targetAccountID string, now time.Time, systemActorID string) (bool, error) {
	actor, err := a.requireMember(actorID)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if err := actor.CanRemoveFrom(a.room, targetAccountID); err != nil {
		return false, stackErr.Error(err)
	}

	targetAccountID = strings.TrimSpace(targetAccountID)
	removedMember, ok := a.members[targetAccountID]
	if !ok || removedMember == nil {
		return false, stackErr.Error(entity.ErrRoomMemberRequired)
	}

	delete(a.members, targetAccountID)
	a.removedMemberIDs = append(a.removedMemberIDs, targetAccountID)
	delete(a.memberUpserts, targetAccountID)
	if err := a.recordEvent(&EventRoomMemberRemoved{
		RoomID:         a.room.ID,
		MemberID:       removedMember.AccountID,
		RoomMemberID:   removedMember.ID,
		MemberName:     removedMember.DisplayName,
		MemberUsername: removedMember.Username,
		MemberRole:     removedMember.Role,
		MemberJoinedAt: removedMember.CreatedAt,
		RemovedAt:      now.UTC(),
	}, now); err != nil {
		return false, stackErr.Error(err)
	}

	a.room.Touch(now)
	a.roomDirty = true
	if _, err := a.appendSystemMessage(systemActorID, fmt.Sprintf("%s left", targetAccountID), now); err != nil {
		return false, stackErr.Error(err)
	}
	return true, nil
}

func (a *RoomAggregate) PinMessage(actorID, messageID string, now time.Time, systemActorID string) error {
	actor, err := a.requireMember(actorID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := actor.CanManageGroup(a.room); err != nil {
		return stackErr.Error(err)
	}
	if err := a.room.PinMessage(messageID, now); err != nil {
		return stackErr.Error(err)
	}

	a.roomDirty = true
	if err := a.recordEvent(&EventRoomMessagePinned{
		RoomID:          a.room.ID,
		PinnedMessageID: a.room.PinnedMessageID,
		PinnedAt:        a.room.UpdatedAt,
	}, now); err != nil {
		return stackErr.Error(err)
	}
	if _, err := a.appendSystemMessage(systemActorID, fmt.Sprintf("message %s pinned", a.room.PinnedMessageID), now); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (a *RoomAggregate) SendMessage(
	messageID,
	senderID string,
	params entity.MessageParams,
	sender MessageSenderIdentity,
	outbox MessageOutboxPayload,
	now time.Time,
) (*entity.MessageEntity, error) {
	if _, err := a.requireMember(senderID); err != nil {
		return nil, stackErr.Error(err)
	}

	message, err := entity.NewMessage(messageID, a.room.ID, senderID, params, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	a.pendingMessages = append(a.pendingMessages, message)
	for _, member := range a.Members() {
		if member == nil || strings.TrimSpace(member.AccountID) == strings.TrimSpace(senderID) {
			continue
		}
		a.pendingReceipts = append(a.pendingReceipts, PendingMessageReceipt{
			MessageID: message.ID,
			AccountID: member.AccountID,
			Status:    "sent",
			CreatedAt: now.UTC(),
			UpdatedAt: now.UTC(),
		})
	}

	a.room.Touch(now)
	a.roomDirty = true
	if err := a.recordMessageCreated(message, sender, outbox, now); err != nil {
		return nil, stackErr.Error(err)
	}

	return message, nil
}

func (a *RoomAggregate) recordMessageCreated(
	message *entity.MessageEntity,
	sender MessageSenderIdentity,
	outbox MessageOutboxPayload,
	now time.Time,
) error {
	return stackErr.Error(a.recordEvent(&sharedevents.RoomMessageCreatedEvent{
		RoomID:                 a.room.ID,
		RoomName:               a.room.Name,
		RoomType:               string(a.room.RoomType),
		MessageID:              message.ID,
		MessageContent:         message.Message,
		MessageType:            message.MessageType,
		ReplyToMessageID:       message.ReplyToMessageID,
		ForwardedFromMessageID: message.ForwardedFromMessageID,
		FileName:               message.FileName,
		FileSize:               message.FileSize,
		MimeType:               message.MimeType,
		ObjectKey:              message.ObjectKey,
		MessageSenderID:        message.SenderID,
		MessageSenderName:      strings.TrimSpace(sender.Name),
		MessageSenderEmail:     strings.TrimSpace(sender.Email),
		MessageSentAt:          message.CreatedAt,
		Mentions:               outbox.Mentions,
		MentionAll:             outbox.MentionAll,
		MentionedAccountIDs:    outbox.MentionedAccountIDs,
	}, now))
}

func (a *RoomAggregate) appendSystemMessage(actorID, body string, now time.Time) (*entity.MessageEntity, error) {
	message, err := entity.NewMessage(newUUID(), a.room.ID, actorID, entity.MessageParams{
		Message:     body,
		MessageType: entity.MessageTypeSystem,
	}, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	a.pendingMessages = append(a.pendingMessages, message)
	a.room.Touch(now)
	a.roomDirty = true
	if err := a.recordMessageCreated(message, MessageSenderIdentity{Name: actorID}, MessageOutboxPayload{}, now); err != nil {
		return nil, stackErr.Error(err)
	}
	return message, nil
}

func (a *RoomAggregate) HasPendingRoomWrite() bool {
	return a.roomDeleted || a.roomDirty || len(a.memberUpserts) > 0 || len(a.removedMemberIDs) > 0 || len(a.pendingMessages) > 0 || len(a.pendingReceipts) > 0 || len(a.Events()) > 0
}

func (a *RoomAggregate) attachMember(member *entity.RoomMemberEntity) error {
	if err := a.attachRestoredMember(member); err != nil {
		return stackErr.Error(err)
	}
	a.memberUpserts[strings.TrimSpace(member.AccountID)] = member
	return nil
}

func (a *RoomAggregate) attachRestoredMember(member *entity.RoomMemberEntity) error {
	if member == nil {
		return stackErr.Error(entity.ErrRoomMemberRequired)
	}

	accountID := strings.TrimSpace(member.AccountID)
	if accountID == "" {
		return stackErr.Error(entity.ErrRoomMemberAccountRequired)
	}
	if _, exists := a.members[accountID]; !exists {
		a.memberOrder = append(a.memberOrder, accountID)
	}

	a.members[accountID] = member
	return nil
}

func (a *RoomAggregate) recordRoomDetailsUpdated(updatedAt time.Time) error {
	return stackErr.Error(a.recordEvent(&EventRoomDetailsUpdated{
		RoomID:      a.room.ID,
		Name:        a.room.Name,
		Description: a.room.Description,
		RoomType:    a.room.RoomType,
		UpdatedAt:   updatedAt,
	}, updatedAt))
}

func (a *RoomAggregate) requireMember(accountID string) (*entity.RoomMemberEntity, error) {
	if a == nil || a.room == nil {
		return nil, stackErr.Error(ErrRoomAggregateNil)
	}

	member, ok := a.members[strings.TrimSpace(accountID)]
	if !ok || member == nil {
		return nil, stackErr.Error(entity.ErrRoomMemberRequired)
	}
	return member, nil
}

func (a *RoomAggregate) recordEvent(payload interface{}, createdAt time.Time) error {
	if err := a.ApplyChange(a, payload); err != nil {
		return stackErr.Error(err)
	}
	events := a.AggregateRoot.Events()
	if len(events) == 0 {
		return nil
	}
	events[len(events)-1].CreatedAt = createdAt.UTC().UnixMilli()
	return nil
}
