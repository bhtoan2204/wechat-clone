package repository

import (
	"context"
	"errors"

	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/modules/relationship/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type followRelationRepo struct {
	db *gorm.DB
}

func newFollowRelationRepo(db *gorm.DB) followRelationStore {
	return &followRelationRepo{db: db}
}

func (r *followRelationRepo) Exists(ctx context.Context, followerID, followeeID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.FollowRelation{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Limit(1).
		Count(&count).Error; err != nil {
		return false, stackErr.Error(err)
	}

	return count > 0, nil
}

func (r *followRelationRepo) Create(ctx context.Context, relation *entity.FollowRelation) error {
	if relation == nil {
		return stackErr.Error(errors.New("follow relation is required"))
	}

	if err := r.db.WithContext(ctx).
		Create(&models.FollowRelation{
			ID:         relation.ID,
			FollowerID: relation.FollowerID,
			FolloweeID: relation.FolloweeID,
			CreatedAt:  relation.CreatedAt,
		}).Error; err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (r *followRelationRepo) Delete(ctx context.Context, followerID, followeeID string) (bool, error) {
	result := r.db.WithContext(ctx).
		Delete(&models.FollowRelation{}, "follower_id = ? AND followee_id = ?", followerID, followeeID)
	if result.Error != nil {
		return false, stackErr.Error(result.Error)
	}

	return result.RowsAffected > 0, nil
}
