package repository

import (
	"context"
	"errors"

	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/modules/relationship/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type relationshipAccountProjectionRepo struct {
	db *gorm.DB
}

func newRelationshipAccountProjectionRepo(db *gorm.DB) repos.RelationshipAccountProjectionRepository {
	return &relationshipAccountProjectionRepo{db: db}
}

func (r *relationshipAccountProjectionRepo) ProjectAccount(ctx context.Context, account *entity.AccountProjection) error {
	if account == nil {
		return stackErr.Error(errors.New("account projection is required"))
	}

	model := &models.RelationshipAccountProjection{
		AccountID:       account.AccountID,
		DisplayName:     account.DisplayName,
		Username:        account.Username,
		AvatarObjectKey: account.AvatarObjectKey,
		CreatedAt:       account.CreatedAt,
		UpdatedAt:       account.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "account_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"display_name":      model.DisplayName,
				"username":          model.Username,
				"avatar_object_key": model.AvatarObjectKey,
				"updated_at":        model.UpdatedAt,
			}),
		}).
		Create(model).Error; err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (r *relationshipAccountProjectionRepo) GetByID(ctx context.Context, accountID string) (*entity.AccountProjection, error) {
	var model models.RelationshipAccountProjection
	if err := r.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		First(&model).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	return &entity.AccountProjection{
		AccountID:       model.AccountID,
		DisplayName:     model.DisplayName,
		Username:        model.Username,
		AvatarObjectKey: model.AvatarObjectKey,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}, nil
}

func (r *relationshipAccountProjectionRepo) Exists(ctx context.Context, accountID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.RelationshipAccountProjection{}).
		Where("account_id = ?", accountID).
		Count(&count).Error; err != nil {
		return false, stackErr.Error(err)
	}

	return count > 0, nil
}
