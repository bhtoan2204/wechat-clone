package repository

import (
	"context"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/notification/domain/repos"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repoImpl struct {
	db     *gorm.DB
	appCtx *appCtx.AppContext

	notificationRepo     repos.NotificationRepository
	pushSubscriptionRepo repos.PushSubscriptionRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) repos.Repos {
	return newRepoImplWithDB(appCtx, appCtx.GetDB())
}

func newRepoImplWithDB(appCtx *appCtx.AppContext, db *gorm.DB) repos.Repos {
	notificationRepo := NewNotificationRepoImpl(db)
	pushSubscriptionRepo := NewPushSubscriptionRepoImpl(db)
	return &repoImpl{
		appCtx: appCtx,
		db:     db,

		notificationRepo:     notificationRepo,
		pushSubscriptionRepo: pushSubscriptionRepo,
	}
}

func (r *repoImpl) NotificationRepository() repos.NotificationRepository {
	return r.notificationRepo
}

func (r *repoImpl) PushSubscriptionRepository() repos.PushSubscriptionRepository {
	return r.pushSubscriptionRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(repos.Repos) error) (err error) {
	log := logging.FromContext(ctx).Named("StartNotificationTransaction")
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
		} else {
			log.Info("transaction committed")
		}
	}()

	err = fn(tr)
	return stackErr.Error(err)
}
