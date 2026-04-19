package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"wechat-clone/core/modules/notification/domain/aggregate"
	"wechat-clone/core/modules/notification/domain/entity"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	"wechat-clone/core/modules/notification/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

type pushSubscriptionRepoImpl struct {
	session *gocql.Session
	tables  models.TableNames
}

type pushSubscriptionRow struct {
	AccountID string
	Endpoint  string
	ID        string
	Keys      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewPushSubscriptionRepoImpl(session *gocql.Session, tables models.TableNames) notificationrepos.PushSubscriptionRepository {
	return &pushSubscriptionRepoImpl{
		session: session,
		tables:  tables,
	}
}

func (r *pushSubscriptionRepoImpl) LoadByAccountAndEndpoint(ctx context.Context, accountID, endpoint string) (*aggregate.PushSubscriptionAggregate, error) {
	accountID = strings.TrimSpace(accountID)
	endpoint = strings.TrimSpace(endpoint)
	if accountID == "" || endpoint == "" {
		return nil, stackErr.Error(notificationrepos.ErrPushSubscriptionNotFound)
	}

	var row pushSubscriptionRow
	err := r.session.Query(
		fmt.Sprintf(`SELECT account_id, endpoint, id, keys, created_at, updated_at FROM %s WHERE account_id = ? AND endpoint = ?`, r.tables.PushSubscriptions),
		accountID,
		endpoint,
	).WithContext(ctx).Scan(
		&row.AccountID,
		&row.Endpoint,
		&row.ID,
		&row.Keys,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, stackErr.Error(notificationrepos.ErrPushSubscriptionNotFound)
	}
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("load push subscription failed: %w", err))
	}

	agg, err := aggregate.NewPushSubscriptionAggregate(row.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.Restore(r.toPushSubscriptionEntity(&row)); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *pushSubscriptionRepoImpl) Save(ctx context.Context, subscription *aggregate.PushSubscriptionAggregate) error {
	snapshot, err := subscription.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(r.session.Query(fmt.Sprintf(`
		INSERT INTO %s (account_id, endpoint, id, keys, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, r.tables.PushSubscriptions),
		snapshot.AccountID,
		snapshot.Endpoint,
		snapshot.ID,
		snapshot.Keys,
		snapshot.CreatedAt.UTC(),
		snapshot.UpdatedAt.UTC(),
	).WithContext(ctx).Exec())
}

func (r *pushSubscriptionRepoImpl) ListPushSubscriptionsByAccountID(ctx context.Context, accountID string) ([]*entity.PushSubscription, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return []*entity.PushSubscription{}, nil
	}

	iter := r.session.Query(
		fmt.Sprintf(`SELECT account_id, endpoint, id, keys, created_at, updated_at FROM %s WHERE account_id = ?`, r.tables.PushSubscriptions),
		accountID,
	).WithContext(ctx).Iter()
	defer iter.Close()

	var row pushSubscriptionRow
	items := make([]*entity.PushSubscription, 0)
	for iter.Scan(&row.AccountID, &row.Endpoint, &row.ID, &row.Keys, &row.CreatedAt, &row.UpdatedAt) {
		copyRow := row
		items = append(items, r.toPushSubscriptionEntity(&copyRow))
		row = pushSubscriptionRow{}
	}
	if err := iter.Close(); err != nil {
		return nil, stackErr.Error(fmt.Errorf("iterate push subscriptions failed: %w", err))
	}
	return items, nil
}

func (r *pushSubscriptionRepoImpl) toPushSubscriptionEntity(row *pushSubscriptionRow) *entity.PushSubscription {
	if row == nil {
		return nil
	}
	return &entity.PushSubscription{
		ID:        row.ID,
		AccountID: row.AccountID,
		Endpoint:  row.Endpoint,
		Keys:      row.Keys,
		CreatedAt: row.CreatedAt.UTC(),
		UpdatedAt: row.UpdatedAt.UTC(),
	}
}
