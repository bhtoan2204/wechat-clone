package aggregate

import (
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/notification/domain/entity"
	"go-socket/core/modules/notification/types"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"

	"github.com/google/uuid"
)

var (
	ErrNotificationAggregateNotInitialized = errors.New("notification aggregate is not initialized")
	ErrNotificationAccountIDRequired       = errors.New("notification account_id is required")
	ErrNotificationTypeRequired            = errors.New("notification type is required")
	ErrNotificationSubjectRequired         = errors.New("notification subject is required")
	ErrNotificationBodyRequired            = errors.New("notification body is required")
	ErrNotificationCreatedAtRequired       = errors.New("notification created_at is required")
	ErrNotificationReadAtRequired          = errors.New("notification read_at is required")
)

type NotificationAggregate struct {
	notification *entity.NotificationEntity
}

func NewNotificationAggregate(notificationID string) (*NotificationAggregate, error) {
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return nil, stackErr.Error(ErrNotificationAggregateNotInitialized)
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
	if a == nil || a.notification == nil || strings.TrimSpace(a.notification.ID) == "" {
		return stackErr.Error(ErrNotificationAggregateNotInitialized)
	}

	accountID = strings.TrimSpace(accountID)
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
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

	normalizedCreatedAt, err := normalizeNotificationTime(createdAt, ErrNotificationCreatedAtRequired)
	if err != nil {
		return stackErr.Error(err)
	}

	a.notification.AccountID = accountID
	a.notification.Type = normalizedType
	a.notification.Subject = subject
	a.notification.Body = body
	a.notification.IsRead = false
	a.notification.ReadAt = nil
	a.notification.CreatedAt = normalizedCreatedAt
	return nil
}

func (a *NotificationAggregate) MarkRead(now time.Time) (bool, error) {
	if a == nil || a.notification == nil {
		return false, stackErr.Error(ErrNotificationAggregateNotInitialized)
	}
	if a.notification.IsRead {
		return false, nil
	}

	readAt, err := normalizeNotificationTime(now, ErrNotificationReadAtRequired)
	if err != nil {
		return false, stackErr.Error(err)
	}

	a.notification.IsRead = true
	a.notification.ReadAt = &readAt
	return true, nil
}

func (a *NotificationAggregate) Snapshot() (*entity.NotificationEntity, error) {
	if a == nil || a.notification == nil {
		return nil, stackErr.Error(ErrNotificationAggregateNotInitialized)
	}

	switch {
	case strings.TrimSpace(a.notification.ID) == "":
		return nil, stackErr.Error(ErrNotificationAggregateNotInitialized)
	case strings.TrimSpace(a.notification.AccountID) == "":
		return nil, stackErr.Error(ErrNotificationAccountIDRequired)
	case strings.TrimSpace(a.notification.Subject) == "":
		return nil, stackErr.Error(ErrNotificationSubjectRequired)
	case strings.TrimSpace(a.notification.Body) == "":
		return nil, stackErr.Error(ErrNotificationBodyRequired)
	}
	if _, err := normalizeNotificationType(a.notification.Type); err != nil {
		return nil, stackErr.Error(err)
	}

	cloned := *a.notification
	cloned.ReadAt = utils.ClonePtr(a.notification.ReadAt)
	return &cloned, nil
}

func WelcomeNotificationID(accountID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("notification:welcome:"+strings.TrimSpace(accountID))).String()
}

func RoomMentionNotificationID(messageID, accountID string) string {
	return uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte("notification:room-mention:"+strings.TrimSpace(messageID)+":"+strings.TrimSpace(accountID)),
	).String()
}

func normalizeNotificationType(value types.NotificationType) (types.NotificationType, error) {
	switch types.NotificationType(strings.TrimSpace(value.String())) {
	case types.NotificationTypeAccountCreated:
		return types.NotificationTypeAccountCreated, nil
	case types.NotificationTypeRoomMention:
		return types.NotificationTypeRoomMention, nil
	default:
		return "", stackErr.Error(ErrNotificationTypeRequired)
	}
}

func normalizeNotificationTime(value time.Time, errRequired error) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, errRequired
	}
	return value.UTC(), nil
}
