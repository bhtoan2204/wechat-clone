package repository

import (
	"context"

	"wechat-clone/core/modules/relationship/domain/entity"
	"wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/modules/relationship/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userRelationshipCounterRepo struct {
	db *gorm.DB
}

func newUserRelationshipCounterRepo(db *gorm.DB) repos.UserRelationshipCounterRepository {
	return &userRelationshipCounterRepo{db: db}
}

func (r *userRelationshipCounterRepo) ApplyDeltas(ctx context.Context, deltas map[string]entity.UserRelationshipCounterDelta) error {
	if len(deltas) == 0 {
		return nil
	}

	stubs := make([]models.UserRelationshipCounters, 0, len(deltas))
	for userID := range deltas {
		if userID == "" {
			continue
		}
		stubs = append(stubs, models.UserRelationshipCounters{UserID: userID})
	}

	if len(stubs) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).
		Create(&stubs).Error; err != nil {
		return stackErr.Error(err)
	}

	for userID, delta := range deltas {
		if userID == "" || delta.IsZero() {
			continue
		}

		if err := r.db.WithContext(ctx).
			Model(&models.UserRelationshipCounters{}).
			Where("user_id = ?", userID).
			Updates(map[string]interface{}{
				"friends_count":     gorm.Expr("friends_count + ?", delta.FriendsCount),
				"followers_count":   gorm.Expr("followers_count + ?", delta.FollowersCount),
				"following_count":   gorm.Expr("following_count + ?", delta.FollowingCount),
				"blocked_count":     gorm.Expr("blocked_count + ?", delta.BlockedCount),
				"pending_in_count":  gorm.Expr("pending_in_count + ?", delta.PendingInCount),
				"pending_out_count": gorm.Expr("pending_out_count + ?", delta.PendingOutCount),
				"updated_at":        gorm.Expr("CURRENT_TIMESTAMP"),
			}).Error; err != nil {
			return stackErr.Error(err)
		}
	}

	return nil
}
