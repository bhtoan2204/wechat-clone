package repository

import (
	"context"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/modules/room/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoomAccountImpl struct {
	db *gorm.DB
}

func NewRoomAccountImpl(db *gorm.DB) repos.RoomAccountRepository {
	return &RoomAccountImpl{db: db}
}

func (r *RoomAccountImpl) ProjectAccount(ctx context.Context, account *entity.AccountEntity) error {
	model := r.toModel(account)

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "account_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"display_name":      account.DisplayName,
				"username":          account.Username,
				"avatar_object_key": account.AvatarObjectKey,
				"updated_at":        account.UpdatedAt,
			}),
		}).
		Create(model).Error; err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (r *RoomAccountImpl) ListByAccountIDs(ctx context.Context, accountIDs []string) ([]*entity.AccountEntity, error) {
	if len(accountIDs) == 0 {
		return []*entity.AccountEntity{}, nil
	}
	var models []models.RoomAccount
	if err := r.db.WithContext(ctx).
		Where("account_id IN ?", accountIDs).
		Find(&models).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	results := make([]*entity.AccountEntity, 0, len(models))
	for _, m := range models {
		model := m
		results = append(results, &entity.AccountEntity{
			AccountID:       model.AccountID,
			DisplayName:     model.DisplayName,
			Username:        model.Username,
			AvatarObjectKey: model.AvatarObjectKey,
			CreatedAt:       model.CreatedAt,
			UpdatedAt:       model.UpdatedAt,
		})
	}

	return results, nil
}

func (r *RoomAccountImpl) toModel(account *entity.AccountEntity) *models.RoomAccount {
	return &models.RoomAccount{
		AccountID:       account.AccountID,
		DisplayName:     account.DisplayName,
		Username:        account.Username,
		AvatarObjectKey: account.AvatarObjectKey,
		CreatedAt:       account.CreatedAt,
		UpdatedAt:       account.UpdatedAt,
	}
}
