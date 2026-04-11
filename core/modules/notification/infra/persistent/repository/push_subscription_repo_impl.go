package repository

import (
	"context"
	"errors"

	"go-socket/core/modules/notification/domain/aggregate"
	"go-socket/core/modules/notification/domain/entity"
	notificationrepos "go-socket/core/modules/notification/domain/repos"
	"go-socket/core/modules/notification/infra/persistent/models"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type pushSubscriptionRepoImpl struct {
	db *gorm.DB
}

func NewPushSubscriptionRepoImpl(db *gorm.DB) notificationrepos.PushSubscriptionRepository {
	return &pushSubscriptionRepoImpl{db: db}
}

func (r *pushSubscriptionRepoImpl) LoadByAccountAndEndpoint(ctx context.Context, accountID, endpoint string) (*aggregate.PushSubscriptionAggregate, error) {
	var existing models.PushSubscriptionModel
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND endpoint = ?", accountID, endpoint).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notificationrepos.ErrPushSubscriptionNotFound
		}
		return nil, stackErr.Error(err)
	}

	agg, err := aggregate.NewPushSubscriptionAggregate(existing.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.Restore(r.toPushSubscriptionEntity(&existing)); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *pushSubscriptionRepoImpl) Save(ctx context.Context, subscription *aggregate.PushSubscriptionAggregate) error {
	snapshot, err := subscription.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "account_id"},
				{Name: "endpoint"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"keys":       snapshot.Keys,
				"updated_at": snapshot.UpdatedAt,
			}),
		}).
		Create(r.toPushSubscriptionModel(snapshot)).Error; err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (r *pushSubscriptionRepoImpl) ListPushSubscriptionsByAccountID(ctx context.Context, accountID string) ([]*entity.PushSubscription, error) {
	var subscriptions []*models.PushSubscriptionModel
	if err := r.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Order("created_at DESC").
		Find(&subscriptions).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	result := make([]*entity.PushSubscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		result = append(result, r.toPushSubscriptionEntity(subscription))
	}

	return result, nil
}

func (r *pushSubscriptionRepoImpl) toPushSubscriptionEntity(m *models.PushSubscriptionModel) *entity.PushSubscription {
	return &entity.PushSubscription{
		ID:        m.ID,
		AccountID: m.AccountID,
		Endpoint:  m.Endpoint,
		Keys:      m.Keys,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (r *pushSubscriptionRepoImpl) toPushSubscriptionModel(e *entity.PushSubscription) *models.PushSubscriptionModel {
	return &models.PushSubscriptionModel{
		ID:        e.ID,
		AccountID: e.AccountID,
		Endpoint:  e.Endpoint,
		Keys:      e.Keys,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
