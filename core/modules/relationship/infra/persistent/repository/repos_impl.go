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

	relationshipPairAggregateRepo  repos.RelationshipPairAggregateRepository
	friendRequestAggregateRepo     repos.FriendRequestAggregateRepository
	friendshipRepo                 repos.FriendshipRepository
	followRelationRepo             repos.FollowRelationRepository
	blockRelationRepo              repos.BlockRelationRepository
	userRelationshipCounterRepo    repos.UserRelationshipCounterRepository
	relationshipAccountProjectRepo repos.RelationshipAccountRepository
	relationshipPairGuardRepo      repos.RelationshipPairGuardRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) repos.Repos {
	return newRepoImplWithDB(appCtx, appCtx.GetDB())
}

func newRepoImplWithDB(appCtx *appCtx.AppContext, db *gorm.DB) repos.Repos {
	relationshipPairAggregateRepo := newRelationshipPairAggregateRepo(db)
	friendRequestAggregateRepo := newFriendRequestAggregateRepo(db)
	friendshipRepo := newFriendshipRepo(db)
	followRelationRepo := newFollowRelationRepo(db)
	blockRelationRepo := newBlockRelationRepo(db)
	userRelationshipCounterRepo := newUserRelationshipCounterRepo(db)
	relationshipAccountProjectRepo := newRelationshipAccountRepo(db)
	relationshipPairGuardRepo := newRelationshipPairGuardRepo(db)
	return &repoImpl{
		appCtx:                         appCtx,
		db:                             db,
		relationshipPairAggregateRepo:  relationshipPairAggregateRepo,
		friendRequestAggregateRepo:     friendRequestAggregateRepo,
		friendshipRepo:                 friendshipRepo,
		followRelationRepo:             followRelationRepo,
		blockRelationRepo:              blockRelationRepo,
		userRelationshipCounterRepo:    userRelationshipCounterRepo,
		relationshipAccountProjectRepo: relationshipAccountProjectRepo,
		relationshipPairGuardRepo:      relationshipPairGuardRepo,
	}
}

func (r *repoImpl) RelationshipPairAggregateRepository() repos.RelationshipPairAggregateRepository {
	return r.relationshipPairAggregateRepo

}

func (r *repoImpl) FriendRequestAggregateRepository() repos.FriendRequestAggregateRepository {
	return r.friendRequestAggregateRepo

}

func (r *repoImpl) FriendshipRepository() repos.FriendshipRepository {
	return r.friendshipRepo
}

func (r *repoImpl) FollowRelationRepository() repos.FollowRelationRepository {
	return r.followRelationRepo
}

func (r *repoImpl) BlockRelationRepository() repos.BlockRelationRepository {
	return r.blockRelationRepo
}

func (r *repoImpl) UserRelationshipCounterRepository() repos.UserRelationshipCounterRepository {
	return r.userRelationshipCounterRepo
}

func (r *repoImpl) RelationshipAccountRepository() repos.RelationshipAccountRepository {
	return r.relationshipAccountProjectRepo
}

func (r *repoImpl) RelationshipPairGuardRepository() repos.RelationshipPairGuardRepository {
	return r.relationshipPairGuardRepo
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
