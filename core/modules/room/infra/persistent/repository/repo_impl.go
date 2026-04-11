package repository

import (
	"context"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repoImpl struct {
	db     *gorm.DB
	appCtx *appCtx.AppContext

	roomRepo        repos.RoomRepository
	messageRepo     repos.MessageRepository
	roomMemberRepo  repos.RoomMemberRepository
	roomOutboxRepo  repos.RoomOutboxEventsRepository
	roomReadRepo    repos.RoomReadRepository
	messageReadRepo repos.MessageReadRepository
	memberReadRepo  repos.RoomMemberReadRepository
	accountRepo     repos.RoomAccountProjectionRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) repos.Repos {
	return newRepoImplWithDB(appCtx, appCtx.GetDB())
}

func newRepoImplWithDB(appCtx *appCtx.AppContext, db *gorm.DB) repos.Repos {
	roomRepo := NewRoomRepoImpl(db, appCtx.GetCache())
	messageRepo := NewMessageRepoImpl(db)
	roomMemberRepo := NewRoomMemberImpl(db)
	roomOutboxRepo := NewRoomOutboxEventsRepoImpl(db)
	roomReadRepo := NewRoomReadRepoImpl(db)
	messageReadRepo := NewMessageReadRepoImpl(db)
	memberReadRepo := NewRoomMemberReadRepoImpl(db)
	accountRepo := NewRoomAccountProjectionImpl(db)

	return &repoImpl{
		db:              db,
		appCtx:          appCtx,
		roomRepo:        roomRepo,
		messageRepo:     messageRepo,
		roomMemberRepo:  roomMemberRepo,
		roomOutboxRepo:  roomOutboxRepo,
		roomReadRepo:    roomReadRepo,
		messageReadRepo: messageReadRepo,
		memberReadRepo:  memberReadRepo,
		accountRepo:     accountRepo,
	}
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

func (r *repoImpl) RoomReadRepository() repos.RoomReadRepository {
	return r.roomReadRepo
}

func (r *repoImpl) MessageReadRepository() repos.MessageReadRepository {
	return r.messageReadRepo
}

func (r *repoImpl) RoomMemberReadRepository() repos.RoomMemberReadRepository {
	return r.memberReadRepo
}

func (r *repoImpl) RoomAccountProjectionRepository() repos.RoomAccountProjectionRepository {
	return r.accountRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(repos.Repos) error) (err error) {
	log := logging.FromContext(ctx).Named("StartRoomTransaction")
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
