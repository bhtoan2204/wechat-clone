package repository

import (
	"context"

	appCtx "go-socket/core/context"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/shared/pkg/logging"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repoImpl struct {
	appCtx *appCtx.AppContext
	db     *gorm.DB

	ledgerRepo  ledgerrepos.LedgerRepository
	paymentRepo ledgerrepos.PaymentRepository
}

func NewRepoImpl(appCtx *appCtx.AppContext) ledgerrepos.Repos {
	return newRepoImplWithDB(appCtx, appCtx.GetDB())
}

func newRepoImplWithDB(appCtx *appCtx.AppContext, db *gorm.DB) ledgerrepos.Repos {
	return &repoImpl{
		appCtx:      appCtx,
		db:          db,
		ledgerRepo:  NewLedgerRepoImpl(db),
		paymentRepo: NewPaymentRepoImpl(db),
	}
}

func (r *repoImpl) LedgerRepository() ledgerrepos.LedgerRepository {
	return r.ledgerRepo
}

func (r *repoImpl) PaymentRepository() ledgerrepos.PaymentRepository {
	return r.paymentRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(ledgerrepos.Repos) error) (err error) {
	log := logging.FromContext(ctx).Named("LedgerTransaction")
	tx := r.db.WithContext(ctx).Begin()
	if beginErr := tx.Error; beginErr != nil {
		log.Errorw("failed to begin transaction", zap.Error(beginErr))
		return beginErr
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
			err = commitErr
			return
		}

		log.Info("transaction committed")
	}()

	err = fn(tr)
	return err
}
