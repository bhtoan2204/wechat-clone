package aggregate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"wechat-clone/core/modules/notification/domain/entity"
	"wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"github.com/google/uuid"
)

var (
	ErrNotificationAggregateNotInitialized = errors.New("notification aggregate is not initialized")
	ErrNotificationIDRequired              = errors.New("notification id is required")
	ErrNotificationAccountIDRequired       = errors.New("notification account_id is required")
	ErrNotificationKindRequired            = errors.New("notification kind is required")
	ErrNotificationTypeRequired            = errors.New("notification type is required")
	ErrNotificationSubjectRequired         = errors.New("notification subject is required")
	ErrNotificationBodyRequired            = errors.New("notification body is required")
	ErrNotificationOccurredAtRequired      = errors.New("notification occurred_at is required")
	ErrNotificationGroupKeyRequired        = errors.New("notification group_key is required")
	ErrNotificationMessageIDRequired       = errors.New("notification message_id is required")
	ErrNotificationMessagePreviewRequired  = errors.New("notification message preview is required")
)

type MessageNotificationInput struct {
	AccountID      string
	GroupKey       string
	Subject        string
	Body           string
	RoomID         string
	RoomName       string
	SenderID       string
	SenderName     string
	MessageID      string
	MessagePreview string
	MessageAt      time.Time
}

type NotificationAggregate struct {
	notification *entity.NotificationEntity
}

func NewNotificationAggregate(notificationID string) (*NotificationAggregate, error) {
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return nil, stackErr.Error(ErrNotificationIDRequired)
	}

	return &NotificationAggregate{
		notification: &entity.NotificationEntity{ID: notificationID},
	}, nil
}

func (a *NotificationAggregate) Restore(snapshot *entity.NotificationEntity) error {
	if snapshot == nil {
		return stackErr.Error(ErrNotificationAggregateNotInitialized)
	}

	cloned := *snapshot
	cloned.ReadAt = utils.ClonePtr(snapshot.ReadAt)
	cloned.LastMessageAt = utils.ClonePtr(snapshot.LastMessageAt)
	a.notification = &cloned
	return nil
}

func (a *NotificationAggregate) Create(
	accountID string,
	notificationType types.NotificationType,
	subject string,
	body string,
	createdAt time.Time,
) error {
	return a.createGeneral(accountID, notificationType, subject, body, createdAt)
}

func (a *NotificationAggregate) createGeneral(
	accountID string,
	notificationType types.NotificationType,
	subject string,
	body string,
	occurredAt time.Time,
) error {
	if err := a.ensureInitialized(); err != nil {
		return stackErr.Error(err)
	}

	accountID = strings.TrimSpace(accountID)
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	occurredAt, err := normalizeNotificationTime(occurredAt)
	if err != nil {
		return stackErr.Error(err)
	}
	normalizedType, err := normalizeNotificationType(notificationType)
	if err != nil {
		return stackErr.Error(err)
	}

	switch {
	case accountID == "":
		return stackErr.Error(ErrNotificationAccountIDRequired)
	case subject == "":
		return stackErr.Error(ErrNotificationSubjectRequired)
	case body == "":
		return stackErr.Error(ErrNotificationBodyRequired)
	}

	a.notification.AccountID = accountID
	a.notification.Kind = types.NotificationKindGeneral
	a.notification.Type = normalizedType
	a.notification.GroupKey = ""
	a.notification.Subject = subject
	a.notification.Body = body
	a.notification.IsRead = false
	a.notification.ReadAt = nil
	a.notification.CreatedAt = occurredAt
	a.notification.UpdatedAt = occurredAt
	a.notification.SortAt = occurredAt
	a.notification.RoomID = ""
	a.notification.RoomName = ""
	a.notification.SenderID = ""
	a.notification.SenderName = ""
	a.notification.MessageCount = 0
	a.notification.LastMessageID = ""
	a.notification.LastMessagePreview = ""
	a.notification.LastMessageAt = nil
	return nil
}

func (a *NotificationAggregate) CreateMessageNotification(input MessageNotificationInput) error {
	if err := a.ensureInitialized(); err != nil {
		return stackErr.Error(err)
	}

	normalized, err := normalizeMessageNotificationInput(input)
	if err != nil {
		return stackErr.Error(err)
	}

	a.notification.AccountID = normalized.AccountID
	a.notification.Kind = types.NotificationKindMessage
	a.notification.Type = types.NotificationTypeRoomMessage
	a.notification.GroupKey = normalized.GroupKey
	a.notification.Subject = normalized.Subject
	a.notification.Body = normalized.Body
	a.notification.IsRead = false
	a.notification.ReadAt = nil
	a.notification.CreatedAt = normalized.MessageAt
	a.notification.UpdatedAt = normalized.MessageAt
	a.notification.SortAt = normalized.MessageAt
	a.notification.RoomID = normalized.RoomID
	a.notification.RoomName = normalized.RoomName
	a.notification.SenderID = normalized.SenderID
	a.notification.SenderName = normalized.SenderName
	a.notification.MessageCount = 1
	a.notification.LastMessageID = normalized.MessageID
	a.notification.LastMessagePreview = normalized.MessagePreview
	a.notification.LastMessageAt = &normalized.MessageAt
	return nil
}

func (a *NotificationAggregate) ApplyMessageActivity(input MessageNotificationInput) (bool, error) {
	if err := a.ensureInitialized(); err != nil {
		return false, stackErr.Error(err)
	}
	if a.notification.Kind.Normalize() != types.NotificationKindMessage {
		return false, stackErr.Error(ErrNotificationKindRequired)
	}

	normalized, err := normalizeMessageNotificationInput(input)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if strings.TrimSpace(a.notification.AccountID) != normalized.AccountID {
		return false, stackErr.Error(fmt.Errorf("notification account_id mismatch"))
	}
	if strings.TrimSpace(a.notification.GroupKey) != normalized.GroupKey {
		return false, stackErr.Error(fmt.Errorf("notification group_key mismatch"))
	}

	// Ignore duplicate or out-of-order deliveries once a newer message is already reflected.
	if a.isMessageActivityStale(normalized.MessageID, normalized.MessageAt) {
		return false, nil
	}

	a.notification.Type = types.NotificationTypeRoomMessage
	a.notification.Subject = normalized.Subject
	a.notification.Body = normalized.Body
	a.notification.RoomID = normalized.RoomID
	a.notification.RoomName = normalized.RoomName
	a.notification.SenderID = normalized.SenderID
	a.notification.SenderName = normalized.SenderName
	a.notification.LastMessageID = normalized.MessageID
	a.notification.LastMessagePreview = normalized.MessagePreview
	a.notification.LastMessageAt = &normalized.MessageAt
	a.notification.UpdatedAt = normalized.MessageAt
	a.notification.SortAt = normalized.MessageAt

	if a.notification.IsRead {
		a.notification.IsRead = false
		a.notification.ReadAt = nil
		a.notification.MessageCount = 1
		return true, nil
	}

	if a.notification.MessageCount <= 0 {
		a.notification.MessageCount = 1
		return true, nil
	}

	a.notification.MessageCount++
	return true, nil
}

func (a *NotificationAggregate) MarkRead(now time.Time) (bool, error) {
	if err := a.ensureInitialized(); err != nil {
		return false, stackErr.Error(err)
	}
	if a.notification.IsRead {
		return false, nil
	}

	readAt, err := normalizeNotificationTime(now)
	if err != nil {
		return false, stackErr.Error(err)
	}

	a.notification.IsRead = true
	a.notification.ReadAt = &readAt
	a.notification.UpdatedAt = readAt
	return true, nil
}

func (a *NotificationAggregate) Snapshot() (*entity.NotificationEntity, error) {
	if err := a.ensureInitialized(); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := validateSnapshot(a.notification); err != nil {
		return nil, stackErr.Error(err)
	}

	cloned := *a.notification
	cloned.ReadAt = utils.ClonePtr(a.notification.ReadAt)
	cloned.LastMessageAt = utils.ClonePtr(a.notification.LastMessageAt)
	return &cloned, nil
}

func (a *NotificationAggregate) ensureInitialized() error {
	if a == nil || a.notification == nil || strings.TrimSpace(a.notification.ID) == "" {
		return stackErr.Error(ErrNotificationAggregateNotInitialized)
	}
	return nil
}

func (a *NotificationAggregate) isMessageActivityStale(messageID string, messageAt time.Time) bool {
	lastMessageID := strings.TrimSpace(a.notification.LastMessageID)
	if lastMessageID == "" {
		return false
	}
	if lastMessageID == strings.TrimSpace(messageID) {
		return true
	}
	if a.notification.LastMessageAt == nil {
		return false
	}
	lastAt := a.notification.LastMessageAt.UTC()
	if messageAt.Before(lastAt) {
		return true
	}
	if messageAt.Equal(lastAt) && strings.TrimSpace(messageID) <= lastMessageID {
		return true
	}
	return false
}

func validateSnapshot(snapshot *entity.NotificationEntity) error {
	switch {
	case strings.TrimSpace(snapshot.ID) == "":
		return stackErr.Error(ErrNotificationIDRequired)
	case strings.TrimSpace(snapshot.AccountID) == "":
		return stackErr.Error(ErrNotificationAccountIDRequired)
	case snapshot.Kind.Normalize() == "":
		return stackErr.Error(ErrNotificationKindRequired)
	case snapshot.SortAt.IsZero():
		return stackErr.Error(ErrNotificationOccurredAtRequired)
	case snapshot.CreatedAt.IsZero():
		return stackErr.Error(ErrNotificationOccurredAtRequired)
	case snapshot.UpdatedAt.IsZero():
		return stackErr.Error(ErrNotificationOccurredAtRequired)
	}

	if _, err := normalizeNotificationType(snapshot.Type); err != nil {
		return stackErr.Error(err)
	}

	switch snapshot.Kind.Normalize() {
	case types.NotificationKindGeneral:
		if strings.TrimSpace(snapshot.Subject) == "" {
			return stackErr.Error(ErrNotificationSubjectRequired)
		}
		if strings.TrimSpace(snapshot.Body) == "" {
			return stackErr.Error(ErrNotificationBodyRequired)
		}
	case types.NotificationKindMessage:
		if strings.TrimSpace(snapshot.GroupKey) == "" {
			return stackErr.Error(ErrNotificationGroupKeyRequired)
		}
		if strings.TrimSpace(snapshot.LastMessageID) == "" {
			return stackErr.Error(ErrNotificationMessageIDRequired)
		}
		if snapshot.LastMessageAt == nil || snapshot.LastMessageAt.IsZero() {
			return stackErr.Error(ErrNotificationOccurredAtRequired)
		}
		if snapshot.MessageCount <= 0 {
			return stackErr.Error(ErrNotificationMessagePreviewRequired)
		}
	default:
		return stackErr.Error(ErrNotificationKindRequired)
	}

	return nil
}

func normalizeNotificationType(value types.NotificationType) (types.NotificationType, error) {
	switch value.Normalize() {
	case types.NotificationTypeAccountCreated:
		return types.NotificationTypeAccountCreated, nil
	case types.NotificationTypeRoomMention:
		return types.NotificationTypeRoomMention, nil
	case types.NotificationTypeRoomMessage:
		return types.NotificationTypeRoomMessage, nil
	case types.NotificationTypeFriendRequestSent:
		return types.NotificationTypeFriendRequestSent, nil
	case types.NotificationTypeFriendRequestCancelled:
		return types.NotificationTypeFriendRequestCancelled, nil
	case types.NotificationTypeFriendRequestAccepted:
		return types.NotificationTypeFriendRequestAccepted, nil
	case types.NotificationTypeFriendRequestRejected:
		return types.NotificationTypeFriendRequestRejected, nil
	default:
		return "", stackErr.Error(ErrNotificationTypeRequired)
	}
}

func normalizeNotificationTime(value time.Time) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, stackErr.Error(ErrNotificationOccurredAtRequired)
	}
	return value.UTC(), nil
}

func normalizeMessageNotificationInput(input MessageNotificationInput) (MessageNotificationInput, error) {
	input.AccountID = strings.TrimSpace(input.AccountID)
	input.GroupKey = strings.TrimSpace(input.GroupKey)
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	input.RoomID = strings.TrimSpace(input.RoomID)
	input.RoomName = strings.TrimSpace(input.RoomName)
	input.SenderID = strings.TrimSpace(input.SenderID)
	input.SenderName = strings.TrimSpace(input.SenderName)
	input.MessageID = strings.TrimSpace(input.MessageID)
	input.MessagePreview = strings.TrimSpace(input.MessagePreview)

	messageAt, err := normalizeNotificationTime(input.MessageAt)
	if err != nil {
		return MessageNotificationInput{}, stackErr.Error(err)
	}
	input.MessageAt = messageAt

	switch {
	case input.AccountID == "":
		return MessageNotificationInput{}, stackErr.Error(ErrNotificationAccountIDRequired)
	case input.GroupKey == "":
		return MessageNotificationInput{}, stackErr.Error(ErrNotificationGroupKeyRequired)
	case input.MessageID == "":
		return MessageNotificationInput{}, stackErr.Error(ErrNotificationMessageIDRequired)
	case input.MessagePreview == "":
		return MessageNotificationInput{}, stackErr.Error(ErrNotificationMessagePreviewRequired)
	case input.Subject == "":
		return MessageNotificationInput{}, stackErr.Error(ErrNotificationSubjectRequired)
	case input.Body == "":
		return MessageNotificationInput{}, stackErr.Error(ErrNotificationBodyRequired)
	}

	return input, nil
}

func WelcomeNotificationID(accountID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("notification:welcome:"+strings.TrimSpace(accountID))).String()
}

func FriendRequestNotificationID(notificationType types.NotificationType, requestID, accountID string) string {
	return uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte("notification:friend-request:"+notificationType.String()+":"+strings.TrimSpace(requestID)+":"+strings.TrimSpace(accountID)),
	).String()
}

func RoomMentionNotificationID(messageID, accountID string) string {
	return uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte("notification:room-mention:"+strings.TrimSpace(messageID)+":"+strings.TrimSpace(accountID)),
	).String()
}

func RoomMessageNotificationID(accountID, groupKey string) string {
	return uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte("notification:room-message:"+strings.TrimSpace(accountID)+":"+strings.TrimSpace(groupKey)),
	).String()
}
