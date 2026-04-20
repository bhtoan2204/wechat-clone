package repository

import (
	"context"
	"errors"

	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/modules/relationship/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type friendshipRepo struct {
	db *gorm.DB
}

func newFriendshipRepo(db *gorm.DB) repos.FriendshipRepository {
	return &friendshipRepo{db: db}
}

func (r *friendshipRepo) ExistsBetween(ctx context.Context, userA, userB string) (bool, error) {
	userLowID, userHighID := normalizePair(userA, userB)

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Friendship{}).
		Where("user_low_id = ? AND user_high_id = ?", userLowID, userHighID).
		Limit(1).
		Count(&count).Error; err != nil {
		return false, stackErr.Error(err)
	}

	return count > 0, nil
}

func (r *friendshipRepo) Create(ctx context.Context, friendship *entity.Friendship) error {
	if friendship == nil {
		return stackErr.Error(errors.New("friendship is required"))
	}

	if err := r.db.WithContext(ctx).
		Create(&models.Friendship{
			ID:                   friendship.ID,
			UserLowID:            friendship.UserLowID,
			UserHighID:           friendship.UserHighID,
			CreatedAt:            friendship.CreatedAt,
			CreatedFromRequestID: friendship.CreatedFromRequestID,
		}).Error; err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (r *friendshipRepo) DeleteBetween(ctx context.Context, userA, userB string) (bool, error) {
	userLowID, userHighID := normalizePair(userA, userB)

	result := r.db.WithContext(ctx).
		Delete(&models.Friendship{}, "user_low_id = ? AND user_high_id = ?", userLowID, userHighID)
	if result.Error != nil {
		return false, stackErr.Error(result.Error)
	}

	return result.RowsAffected > 0, nil
}
