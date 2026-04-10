package repository

import (
	"context"
	"go-socket/core/modules/notification/application/dto/out"
	notificationquery "go-socket/core/modules/notification/application/query"
	"go-socket/core/modules/notification/domain/entity"
	"go-socket/core/modules/notification/domain/repos"
	"go-socket/core/modules/notification/infra/persistent/models"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"
	"time"

	"gorm.io/gorm"
)

type notificationRepoImpl struct {
	db *gorm.DB
}

var _ repos.NotificationRepository = (*notificationRepoImpl)(nil)
var _ notificationquery.NotificationReadRepository = (*notificationRepoImpl)(nil)

func NewNotificationRepoImpl(db *gorm.DB) *notificationRepoImpl {
	return &notificationRepoImpl{db: db}
}

func NewNotificationReadRepository(db *gorm.DB) notificationquery.NotificationReadRepository {
	return NewNotificationRepoImpl(db)
}

func (r *notificationRepoImpl) CreateNotification(ctx context.Context, notification *entity.NotificationEntity) error {
	m := r.toModel(notification)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *notificationRepoImpl) ListNotifications(ctx context.Context, options utils.QueryOptions) ([]*out.NotificationResponse, error) {
	var notifications []*models.NotificationModel

	tx := r.db.WithContext(ctx).Model(&models.NotificationModel{})

	for _, condition := range options.Conditions {
		switch condition.Operator {
		case utils.IsNull, utils.IsNotNull:
			tx = tx.Where(condition.BuildCondition())
		case utils.Raw:
			if condition.Value == nil {
				tx = tx.Where(condition.BuildCondition())
				break
			}
			if values, ok := condition.Value.([]interface{}); ok {
				tx = tx.Where(condition.BuildCondition(), values...)
				break
			}
			tx = tx.Where(condition.BuildCondition(), condition.Value)
		default:
			tx = tx.Where(condition.BuildCondition(), condition.Value)
		}
	}

	if options.OrderBy != "" && options.OrderDirection != "" {
		tx = tx.Order(options.OrderBy + " " + options.OrderDirection)
	} else {
		tx = tx.Order("created_at DESC").Order("id DESC")
	}
	if options.Limit != nil {
		tx = tx.Limit(*options.Limit)
	}
	if options.Offset != nil {
		tx = tx.Offset(*options.Offset)
	}

	if err := tx.Find(&notifications).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	responses := make([]*out.NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		responses = append(responses, &out.NotificationResponse{
			ID:        notification.ID,
			AccountID: notification.AccountID,
			Type:      notification.Type,
			Subject:   notification.Subject,
			Body:      notification.Body,
			IsRead:    notification.IsRead,
			ReadAt:    notification.ReadAt.Format(time.DateTime),
			CreatedAt: notification.CreatedAt.Format(time.DateTime),
		})
	}

	return responses, nil
}

func (r *notificationRepoImpl) toModel(e *entity.NotificationEntity) *models.NotificationModel {
	return &models.NotificationModel{
		ID:        e.ID,
		AccountID: e.AccountID,
		Type:      e.Type.String(),
		Subject:   e.Subject,
		Body:      e.Body,
		IsRead:    e.IsRead,
		ReadAt:    e.ReadAt,
		CreatedAt: e.CreatedAt,
	}
}
