package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"wechat-clone/core/modules/notification/domain/aggregate"
	"wechat-clone/core/modules/notification/domain/entity"
	"wechat-clone/core/modules/notification/domain/repos"
	"wechat-clone/core/modules/notification/infra/persistent/models"
	notificationtypes "wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

type notificationByIDRow struct {
	ID                 string
	AccountID          string
	Kind               string
	Type               string
	GroupKey           string
	Subject            string
	Body               string
	IsRead             bool
	ReadAt             *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SortAt             time.Time
	RoomID             string
	RoomName           string
	SenderID           string
	SenderName         string
	MessageCount       int
	LastMessageID      string
	LastMessagePreview string
	LastMessageAt      *time.Time
}

type notificationByAccountRow = notificationByIDRow

type messageNotificationGroupRow struct {
	AccountID          string
	GroupKey           string
	NotificationID     string
	IsRead             bool
	ReadAt             *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SortAt             time.Time
	RoomID             string
	RoomName           string
	SenderID           string
	SenderName         string
	MessageCount       int
	LastMessageID      string
	LastMessagePreview string
	LastMessageAt      *time.Time
	Subject            string
	Body               string
}

type notificationRepoImpl struct {
	session *gocql.Session
	tables  models.TableNames
}

var _ repos.NotificationRepository = (*notificationRepoImpl)(nil)

func newNotificationRepo(session *gocql.Session) (*notificationRepoImpl, error) {
	if session == nil {
		return nil, stackErr.Error(fmt.Errorf("notification repository requires cassandra session"))
	}
	return &notificationRepoImpl{
		session: session,
		tables:  models.DefaultTableNames(),
	}, nil
}

func (r *notificationRepoImpl) Load(ctx context.Context, notificationID string) (*aggregate.NotificationAggregate, error) {
	row, err := r.getNotificationByID(ctx, notificationID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := aggregate.NewNotificationAggregate(row.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.Restore(r.byIDRowToEntity(row)); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *notificationRepoImpl) LoadMessageGroup(ctx context.Context, accountID, groupKey string) (*aggregate.NotificationAggregate, error) {
	row, err := r.getMessageGroupRow(ctx, accountID, groupKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := aggregate.NewNotificationAggregate(row.NotificationID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.Restore(r.messageGroupRowToEntity(row)); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *notificationRepoImpl) Save(ctx context.Context, notification *aggregate.NotificationAggregate) error {
	snapshot, err := notification.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}

	previous, err := r.getNotificationByID(ctx, snapshot.ID)
	if err != nil && !errors.Is(err, repos.ErrNotificationNotFound) {
		return stackErr.Error(err)
	}

	if previous != nil && previous.AccountID == snapshot.AccountID && !previous.SortAt.Equal(snapshot.SortAt) {
		if err := r.deleteNotificationByAccount(ctx, previous.AccountID, previous.SortAt, previous.ID); err != nil {
			return stackErr.Error(err)
		}
	}

	if err := r.upsertNotificationByID(ctx, snapshot); err != nil {
		return stackErr.Error(err)
	}
	if err := r.upsertNotificationByAccount(ctx, snapshot); err != nil {
		return stackErr.Error(err)
	}
	if err := r.syncUnreadIndex(ctx, previous, snapshot); err != nil {
		return stackErr.Error(err)
	}

	if snapshot.Kind.Normalize() == notificationtypes.NotificationKindMessage {
		if err := r.upsertMessageGroup(ctx, snapshot); err != nil {
			return stackErr.Error(err)
		}
	}

	return nil
}

func (r *notificationRepoImpl) ListByAccountID(ctx context.Context, accountID string, cursor *repos.NotificationListCursor, limit int) ([]*entity.NotificationEntity, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return []*entity.NotificationEntity{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	var (
		iter *gocql.Iter
		row  notificationByAccountRow
	)

	query := fmt.Sprintf(`
		SELECT id, account_id, kind, type, group_key, subject, body, is_read, read_at, created_at, updated_at, sort_at,
		       room_id, room_name, sender_id, sender_name, message_count, last_message_id, last_message_preview, last_message_at
		FROM %s
		WHERE account_id = ?
	`, r.tables.NotificationByAccount)

	if cursor != nil && !cursor.SortAt.IsZero() && strings.TrimSpace(cursor.NotificationID) != "" {
		query += " AND (sort_at, id) < (?, ?)"
		iter = r.session.Query(query, accountID, cursor.SortAt.UTC(), strings.TrimSpace(cursor.NotificationID)).
			WithContext(ctx).
			PageSize(limit).
			Iter()
	} else {
		iter = r.session.Query(query, accountID).WithContext(ctx).PageSize(limit).Iter()
	}

	defer iter.Close()

	items := make([]*entity.NotificationEntity, 0, limit)
	for iter.Scan(
		&row.ID,
		&row.AccountID,
		&row.Kind,
		&row.Type,
		&row.GroupKey,
		&row.Subject,
		&row.Body,
		&row.IsRead,
		&row.ReadAt,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.SortAt,
		&row.RoomID,
		&row.RoomName,
		&row.SenderID,
		&row.SenderName,
		&row.MessageCount,
		&row.LastMessageID,
		&row.LastMessagePreview,
		&row.LastMessageAt,
	) {
		copyRow := row
		items = append(items, r.byIDRowToEntity(&copyRow))
		row = notificationByAccountRow{}
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate notifications by account failed: %w", err))
	}
	return items, nil
}

func (r *notificationRepoImpl) ListUnreadByAccountID(ctx context.Context, accountID string, limit int) ([]*entity.NotificationEntity, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return []*entity.NotificationEntity{}, nil
	}

	query := fmt.Sprintf(`SELECT notification_id FROM %s WHERE account_id = ?`, r.tables.NotificationUnreadIndex)
	iter := r.session.Query(query, accountID).WithContext(ctx).Iter()
	defer iter.Close()

	var notificationID string
	items := make([]*entity.NotificationEntity, 0)
	for iter.Scan(&notificationID) {
		agg, err := r.Load(ctx, notificationID)
		if err != nil {
			if errors.Is(err, repos.ErrNotificationNotFound) {
				notificationID = ""
				continue
			}
			return nil, stackErr.Error(err)
		}
		snapshot, err := agg.Snapshot()
		if err != nil {
			return nil, stackErr.Error(err)
		}
		items = append(items, snapshot)
		if limit > 0 && len(items) >= limit {
			break
		}
		notificationID = ""
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate unread notification ids failed: %w", err))
	}
	return items, nil
}

func (r *notificationRepoImpl) CountUnread(ctx context.Context, accountID string) (int, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return 0, nil
	}

	iter := r.session.Query(
		fmt.Sprintf(`SELECT notification_id FROM %s WHERE account_id = ?`, r.tables.NotificationUnreadIndex),
		accountID,
	).WithContext(ctx).Iter()
	defer iter.Close()

	var (
		notificationID string
		count          int
	)
	for iter.Scan(&notificationID) {
		count++
		notificationID = ""
	}
	if err := iter.Close(); err != nil {
		return 0, stackErr.Error(fmt.Errorf("iterate unread notification count failed: %w", err))
	}
	return count, nil
}

func (r *notificationRepoImpl) getNotificationByID(ctx context.Context, notificationID string) (*notificationByIDRow, error) {
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return nil, stackErr.Error(repos.ErrNotificationNotFound)
	}

	var row notificationByIDRow
	err := r.session.Query(fmt.Sprintf(`
		SELECT id, account_id, kind, type, group_key, subject, body, is_read, read_at, created_at, updated_at, sort_at,
		       room_id, room_name, sender_id, sender_name, message_count, last_message_id, last_message_preview, last_message_at
		FROM %s
		WHERE id = ?
	`, r.tables.NotificationByID), notificationID).WithContext(ctx).Scan(
		&row.ID,
		&row.AccountID,
		&row.Kind,
		&row.Type,
		&row.GroupKey,
		&row.Subject,
		&row.Body,
		&row.IsRead,
		&row.ReadAt,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.SortAt,
		&row.RoomID,
		&row.RoomName,
		&row.SenderID,
		&row.SenderName,
		&row.MessageCount,
		&row.LastMessageID,
		&row.LastMessagePreview,
		&row.LastMessageAt,
	)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, stackErr.Error(repos.ErrNotificationNotFound)
	}
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("get notification by id failed: %w", err))
	}
	return &row, nil
}

func (r *notificationRepoImpl) getMessageGroupRow(ctx context.Context, accountID, groupKey string) (*messageNotificationGroupRow, error) {
	accountID = strings.TrimSpace(accountID)
	groupKey = strings.TrimSpace(groupKey)
	if accountID == "" || groupKey == "" {
		return nil, stackErr.Error(repos.ErrNotificationNotFound)
	}

	var row messageNotificationGroupRow
	err := r.session.Query(fmt.Sprintf(`
		SELECT account_id, group_key, notification_id, is_read, read_at, created_at, updated_at, sort_at,
		       room_id, room_name, sender_id, sender_name, message_count, last_message_id, last_message_preview, last_message_at,
		       subject, body
		FROM %s
		WHERE account_id = ? AND group_key = ?
	`, r.tables.MessageNotificationGroups), accountID, groupKey).WithContext(ctx).Scan(
		&row.AccountID,
		&row.GroupKey,
		&row.NotificationID,
		&row.IsRead,
		&row.ReadAt,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.SortAt,
		&row.RoomID,
		&row.RoomName,
		&row.SenderID,
		&row.SenderName,
		&row.MessageCount,
		&row.LastMessageID,
		&row.LastMessagePreview,
		&row.LastMessageAt,
		&row.Subject,
		&row.Body,
	)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, stackErr.Error(repos.ErrNotificationNotFound)
	}
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("get message notification group failed: %w", err))
	}
	return &row, nil
}

func (r *notificationRepoImpl) upsertNotificationByID(ctx context.Context, snapshot *entity.NotificationEntity) error {
	return stackErr.Error(r.session.Query(fmt.Sprintf(`
		INSERT INTO %s (
			id, account_id, kind, type, group_key, subject, body, is_read, read_at, created_at, updated_at, sort_at,
			room_id, room_name, sender_id, sender_name, message_count, last_message_id, last_message_preview, last_message_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.tables.NotificationByID),
		snapshot.ID,
		snapshot.AccountID,
		snapshot.Kind.String(),
		snapshot.Type.String(),
		snapshot.GroupKey,
		snapshot.Subject,
		snapshot.Body,
		snapshot.IsRead,
		snapshot.ReadAt,
		snapshot.CreatedAt.UTC(),
		snapshot.UpdatedAt.UTC(),
		snapshot.SortAt.UTC(),
		snapshot.RoomID,
		snapshot.RoomName,
		snapshot.SenderID,
		snapshot.SenderName,
		snapshot.MessageCount,
		snapshot.LastMessageID,
		snapshot.LastMessagePreview,
		snapshot.LastMessageAt,
	).WithContext(ctx).Exec())
}

func (r *notificationRepoImpl) upsertNotificationByAccount(ctx context.Context, snapshot *entity.NotificationEntity) error {
	return stackErr.Error(r.session.Query(fmt.Sprintf(`
		INSERT INTO %s (
			account_id, sort_at, id, kind, type, group_key, subject, body, is_read, read_at, created_at, updated_at,
			room_id, room_name, sender_id, sender_name, message_count, last_message_id, last_message_preview, last_message_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.tables.NotificationByAccount),
		snapshot.AccountID,
		snapshot.SortAt.UTC(),
		snapshot.ID,
		snapshot.Kind.String(),
		snapshot.Type.String(),
		snapshot.GroupKey,
		snapshot.Subject,
		snapshot.Body,
		snapshot.IsRead,
		snapshot.ReadAt,
		snapshot.CreatedAt.UTC(),
		snapshot.UpdatedAt.UTC(),
		snapshot.RoomID,
		snapshot.RoomName,
		snapshot.SenderID,
		snapshot.SenderName,
		snapshot.MessageCount,
		snapshot.LastMessageID,
		snapshot.LastMessagePreview,
		snapshot.LastMessageAt,
	).WithContext(ctx).Exec())
}

func (r *notificationRepoImpl) deleteNotificationByAccount(ctx context.Context, accountID string, sortAt time.Time, notificationID string) error {
	return stackErr.Error(r.session.Query(
		fmt.Sprintf(`DELETE FROM %s WHERE account_id = ? AND sort_at = ? AND id = ?`, r.tables.NotificationByAccount),
		strings.TrimSpace(accountID),
		sortAt.UTC(),
		strings.TrimSpace(notificationID),
	).WithContext(ctx).Exec())
}

func (r *notificationRepoImpl) syncUnreadIndex(ctx context.Context, previous *notificationByIDRow, current *entity.NotificationEntity) error {
	if current == nil {
		return nil
	}

	if current.IsRead {
		if previous != nil && !previous.IsRead {
			return stackErr.Error(r.deleteUnreadIndex(ctx, current.AccountID, current.ID))
		}
		return nil
	}

	if err := r.insertUnreadIndex(ctx, current.AccountID, current.ID); err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *notificationRepoImpl) insertUnreadIndex(ctx context.Context, accountID, notificationID string) error {
	return stackErr.Error(r.session.Query(
		fmt.Sprintf(`INSERT INTO %s (account_id, notification_id) VALUES (?, ?)`, r.tables.NotificationUnreadIndex),
		strings.TrimSpace(accountID),
		strings.TrimSpace(notificationID),
	).WithContext(ctx).Exec())
}

func (r *notificationRepoImpl) deleteUnreadIndex(ctx context.Context, accountID, notificationID string) error {
	return stackErr.Error(r.session.Query(
		fmt.Sprintf(`DELETE FROM %s WHERE account_id = ? AND notification_id = ?`, r.tables.NotificationUnreadIndex),
		strings.TrimSpace(accountID),
		strings.TrimSpace(notificationID),
	).WithContext(ctx).Exec())
}

func (r *notificationRepoImpl) upsertMessageGroup(ctx context.Context, snapshot *entity.NotificationEntity) error {
	return stackErr.Error(r.session.Query(fmt.Sprintf(`
		INSERT INTO %s (
			account_id, group_key, notification_id, is_read, read_at, created_at, updated_at, sort_at,
			room_id, room_name, sender_id, sender_name, message_count, last_message_id, last_message_preview, last_message_at,
			subject, body
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.tables.MessageNotificationGroups),
		snapshot.AccountID,
		snapshot.GroupKey,
		snapshot.ID,
		snapshot.IsRead,
		snapshot.ReadAt,
		snapshot.CreatedAt.UTC(),
		snapshot.UpdatedAt.UTC(),
		snapshot.SortAt.UTC(),
		snapshot.RoomID,
		snapshot.RoomName,
		snapshot.SenderID,
		snapshot.SenderName,
		snapshot.MessageCount,
		snapshot.LastMessageID,
		snapshot.LastMessagePreview,
		snapshot.LastMessageAt,
		snapshot.Subject,
		snapshot.Body,
	).WithContext(ctx).Exec())
}

func (r *notificationRepoImpl) byIDRowToEntity(row *notificationByIDRow) *entity.NotificationEntity {
	if row == nil {
		return nil
	}

	return &entity.NotificationEntity{
		ID:                 row.ID,
		AccountID:          row.AccountID,
		Kind:               notificationtypes.NotificationKind(row.Kind),
		Type:               notificationtypes.NotificationType(row.Type),
		GroupKey:           row.GroupKey,
		Subject:            row.Subject,
		Body:               row.Body,
		IsRead:             row.IsRead,
		ReadAt:             cloneOptionalTime(row.ReadAt),
		CreatedAt:          row.CreatedAt.UTC(),
		UpdatedAt:          row.UpdatedAt.UTC(),
		SortAt:             row.SortAt.UTC(),
		RoomID:             row.RoomID,
		RoomName:           row.RoomName,
		SenderID:           row.SenderID,
		SenderName:         row.SenderName,
		MessageCount:       row.MessageCount,
		LastMessageID:      row.LastMessageID,
		LastMessagePreview: row.LastMessagePreview,
		LastMessageAt:      cloneOptionalTime(row.LastMessageAt),
	}
}

func (r *notificationRepoImpl) messageGroupRowToEntity(row *messageNotificationGroupRow) *entity.NotificationEntity {
	if row == nil {
		return nil
	}

	return &entity.NotificationEntity{
		ID:                 row.NotificationID,
		AccountID:          row.AccountID,
		Kind:               notificationtypes.NotificationKindMessage,
		Type:               notificationtypes.NotificationTypeRoomMessage,
		GroupKey:           row.GroupKey,
		Subject:            row.Subject,
		Body:               row.Body,
		IsRead:             row.IsRead,
		ReadAt:             cloneOptionalTime(row.ReadAt),
		CreatedAt:          row.CreatedAt.UTC(),
		UpdatedAt:          row.UpdatedAt.UTC(),
		SortAt:             row.SortAt.UTC(),
		RoomID:             row.RoomID,
		RoomName:           row.RoomName,
		SenderID:           row.SenderID,
		SenderName:         row.SenderName,
		MessageCount:       row.MessageCount,
		LastMessageID:      row.LastMessageID,
		LastMessagePreview: row.LastMessagePreview,
		LastMessageAt:      cloneOptionalTime(row.LastMessageAt),
	}
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}
