package repository

import (
	"context"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"wechat-clone/core/modules/relationship/domain/repos"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repoImpl struct {
	appCtx *appCtx.AppContext
	db     *gorm.DB

	relationshipPairAggregateRepo repos.RelationshipPairAggregateRepository
	friendRequestAggregateRepo    repos.FriendRequestAggregateRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) repos.Repos {
	return newRepoImplWithDB(appCtx, appCtx.GetDB())
}

func newRepoImplWithDB(appCtx *appCtx.AppContext, db *gorm.DB) repos.Repos {
	return &repoImpl{
		appCtx:                        appCtx,
		db:                            db,
		relationshipPairAggregateRepo: newRelationshipPairAggregateRepo(db),
		friendRequestAggregateRepo:    newFriendRequestAggregateRepo(db),
	}
}

func (r *repoImpl) RelationshipPairAggregateRepository() repos.RelationshipPairAggregateRepository {
	return r.relationshipPairAggregateRepo
}

func (r *repoImpl) FriendRequestAggregateRepository() repos.FriendRequestAggregateRepository {
	return r.friendRequestAggregateRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(repos.Repos) error) (err error) {
	log := logging.FromContext(ctx).Named("RelationshipTransaction")
	tx := r.db.WithContext(ctx).Begin()
	if beginErr := tx.Error; beginErr != nil {
		log.Errorw("failed to begin transaction", zap.Error(beginErr))
		return stackErr.Error(beginErr)
	}

	tr := newRepoImplWithDB(r.appCtx, tx)

	defer func() {
		if rec := recover(); rec != nil {
			_ = tx.Rollback().Error
			log.Errorw("panic -> rollback", zap.Any("panic", rec))
			panic(rec)
		}

		if err != nil {
			_ = tx.Rollback().Error
			log.Errorw("transaction rollback", zap.Error(err))
			return
		}

		if commitErr := tx.Commit().Error; commitErr != nil {
			log.Errorw("commit failed", zap.Error(commitErr))
			err = stackErr.Error(commitErr)
			return
		}

		log.Info("transaction committed")
	}()

	err = fn(tr)
	return stackErr.Error(err)
}
