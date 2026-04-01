package repository

import (
	"context"
	"go-socket/core/modules/payment/domain/entity"
	"go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/model"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"

	"gorm.io/gorm"
)

type paymentHistoryRepoImpl struct {
	db *gorm.DB
}

func NewPaymentHistoryRepoImpl(db *gorm.DB) repos.PaymentHistoryRepository {
	return &paymentHistoryRepoImpl{
		db: db,
	}
}

func (r *paymentHistoryRepoImpl) ListPaymentHistory(ctx context.Context, options utils.QueryOptions) ([]*entity.PaymentHistory, error) {
	var rows []*model.PaymentHistoryModel

	tx := r.db.WithContext(ctx).Model(&model.PaymentHistoryModel{})

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

	if err := tx.Find(&rows).Error; err != nil {
		return nil, stackerr.Error(err)
	}

	result := make([]*entity.PaymentHistory, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}

		history := &entity.PaymentHistory{
			ID:        row.ID,
			Type:      row.Type,
			Amount:    row.Amount,
			Balance:   row.Balance,
			CreatedAt: row.CreatedAt,
		}
		if row.SenderID != nil {
			history.SenderID = *row.SenderID
		}
		if row.ReceiverID != nil {
			history.ReceiverID = *row.ReceiverID
		}
		if row.SenderName != nil {
			history.SenderName = *row.SenderName
		}
		if row.ReceiverName != nil {
			history.ReceiverName = *row.ReceiverName
		}
		result = append(result, history)
	}

	return result, nil
}
