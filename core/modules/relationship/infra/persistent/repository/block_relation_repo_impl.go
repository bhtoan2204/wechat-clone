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

type blockRelationRepo struct {
	db *gorm.DB
}

func newBlockRelationRepo(db *gorm.DB) repos.BlockRelationRepository {
	return &blockRelationRepo{db: db}
}

func (r *blockRelationRepo) Exists(ctx context.Context, blockerID, blockedID string) (bool, error) {
	return r.existsWhere(ctx, "blocker_id = ? AND blocked_id = ?", blockerID, blockedID)
}

func (r *blockRelationRepo) ExistsAnyDirection(ctx context.Context, userA, userB string) (bool, error) {
	return r.existsWhere(
		ctx,
		"(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
		userA,
		userB,
		userB,
		userA,
	)
}

func (r *blockRelationRepo) Create(ctx context.Context, relation *entity.BlockRelation) error {
	if relation == nil {
		return stackErr.Error(errors.New("block relation is required"))
	}

	if err := r.db.WithContext(ctx).
		Create(&models.BlockRelation{
			ID:        relation.ID,
			BlockerID: relation.BlockerID,
			BlockedID: relation.BlockedID,
			Reason:    relation.Reason,
			CreatedAt: relation.CreatedAt,
		}).Error; err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (r *blockRelationRepo) Delete(ctx context.Context, blockerID, blockedID string) (bool, error) {
	result := r.db.WithContext(ctx).
		Delete(&models.BlockRelation{}, "blocker_id = ? AND blocked_id = ?", blockerID, blockedID)
	if result.Error != nil {
		return false, stackErr.Error(result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (r *blockRelationRepo) existsWhere(ctx context.Context, query string, args ...interface{}) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.BlockRelation{}).
		Where(query, args...).
		Limit(1).
		Count(&count).Error; err != nil {
		return false, stackErr.Error(err)
	}

	return count > 0, nil
}
