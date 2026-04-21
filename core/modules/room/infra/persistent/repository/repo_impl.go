package repository

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repoImpl struct {
	db     *gorm.DB
	appCtx *appCtx.AppContext

	roomAggregateRepo repos.RoomAggregateRepository
	messageAggRepo    repos.MessageAggregateRepository

	roomRepo       repos.RoomRepository
	messageRepo    repos.MessageRepository
	roomMemberRepo repos.RoomMemberRepository
	roomOutboxRepo repos.RoomOutboxEventsRepository
	accountRepo    repos.RoomAccountRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) (repos.Repos, error) {
	return newRepoImplWithDB(appCtx, appCtx.GetDB())
}

func newRepoImplWithDB(appCtx *appCtx.AppContext, db *gorm.DB) (repos.Repos, error) {
	roomRepo := NewRoomRepoImpl(db, appCtx.GetCache())
	messageRepo := NewMessageRepoImpl(db)
	roomMemberRepo := NewRoomMemberImpl(db)
	roomOutboxRepo := NewRoomOutboxEventsRepoImpl(db)
	accountRepo := NewRoomAccountImpl(db)
	roomAggregateRepo := newRoomAggregateRepoImpl(db, roomRepo, roomMemberRepo, messageRepo, roomOutboxRepo, accountRepo)
	messageAggregateRepo := newMessageAggregateRepoImpl(db, messageRepo, roomRepo, roomMemberRepo, accountRepo, roomOutboxRepo)

	return &repoImpl{
		roomAggregateRepo: roomAggregateRepo,
		messageAggRepo:    messageAggregateRepo,
		db:                db,
		appCtx:            appCtx,
		roomRepo:          roomRepo,
		messageRepo:       messageRepo,
		roomMemberRepo:    roomMemberRepo,
		roomOutboxRepo:    roomOutboxRepo,
		accountRepo:       accountRepo,
	}, nil
}

func (r *repoImpl) RoomAggregateRepository() repos.RoomAggregateRepository {
	return r.roomAggregateRepo
}

func (r *repoImpl) MessageAggregateRepository() repos.MessageAggregateRepository {
	return r.messageAggRepo
}

func (r *repoImpl) RoomRepository() repos.RoomRepository {
	return r.roomRepo
}

func (r *repoImpl) MessageRepository() repos.MessageRepository {
	return r.messageRepo
}

func (r *repoImpl) RoomMemberRepository() repos.RoomMemberRepository {
	return r.roomMemberRepo
}

func (r *repoImpl) RoomOutboxEventsRepository() repos.RoomOutboxEventsRepository {
	return r.roomOutboxRepo
}

func (r *repoImpl) RoomAccountRepository() repos.RoomAccountRepository {
	return r.accountRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(repos.Repos) error) (err error) {
	log := logging.FromContext(ctx).Named("StartRoomTransaction")
	tx := r.db.WithContext(ctx).Begin()
	if beginErr := tx.Error; beginErr != nil {
		log.Errorw("failed to begin transaction", zap.Error(beginErr))
		return stackErr.Error(beginErr)
	}

	tr, buildErr := newRepoImplWithDB(r.appCtx, tx)
	if buildErr != nil {
		_ = tx.Rollback().Error
		return stackErr.Error(buildErr)
	}

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
