package repository

import (
	"context"
	"errors"

	"wechat-clone/core/modules/relationship/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type relationshipPairGuardRepo struct {
	db *gorm.DB
}

func newRelationshipPairGuardRepo(db *gorm.DB) relationshipPairGuardStore {
	return &relationshipPairGuardRepo{db: db}
}

func (r *relationshipPairGuardRepo) LockPair(ctx context.Context, userA, userB string) error {
	if userA == "" || userB == "" {
		return stackErr.Error(errors.New("pair users are required"))
	}

	userLowID, userHighID := normalizePair(userA, userB)
	guard := &models.RelationshipPairGuard{
		UserLowID:  userLowID,
		UserHighID: userHighID,
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_low_id"}, {Name: "user_high_id"}},
			DoNothing: true,
		}).
		Create(guard).Error; err != nil {
		return stackErr.Error(err)
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_low_id = ? AND user_high_id = ?", userLowID, userHighID).
		First(&models.RelationshipPairGuard{}).Error; err != nil {
		return stackErr.Error(err)
	}

	return nil
}
