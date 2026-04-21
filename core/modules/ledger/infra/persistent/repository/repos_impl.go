package repository

import (
	"context"

	appCtx "wechat-clone/core/context"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repoImpl struct {
	appCtx *appCtx.AppContext
	db     *gorm.DB

	ledgerAccountAggregateRepo ledgerrepos.LedgerAccountAggregateRepository
	ledgerOutboxEventsRepo     ledgerrepos.LedgerOutboxEventsRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) ledgerrepos.Repos {
	return newRepoImplWithDB(appCtx, appCtx.GetDB())
}

func newRepoImplWithDB(appCtx *appCtx.AppContext, db *gorm.DB) ledgerrepos.Repos {
	ledgerAccountRepo := newLedgerAccountAggregateRepoImpl(db)
	return &repoImpl{
		appCtx:                     appCtx,
		db:                         db,
		ledgerAccountAggregateRepo: ledgerAccountRepo,
		ledgerOutboxEventsRepo:     NewLedgerOutboxEventsRepoImpl(db),
	}
}

func (r *repoImpl) LedgerAccountAggregateRepository() ledgerrepos.LedgerAccountAggregateRepository {
	return r.ledgerAccountAggregateRepo
}

func (r *repoImpl) LedgerOutboxEventsRepository() ledgerrepos.LedgerOutboxEventsRepository {
	return r.ledgerOutboxEventsRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(ledgerrepos.Repos) error) (err error) {
	log := logging.FromContext(ctx).Named("LedgerTransaction")
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
