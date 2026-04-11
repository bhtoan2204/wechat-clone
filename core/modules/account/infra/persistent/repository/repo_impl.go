package repos

import (
	"context"
	"go-socket/core/modules/account/domain/repos"
	sharedcache "go-socket/core/shared/infra/cache"
	"go-socket/core/shared/pkg/logging"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type repoImpl struct {
	db    *gorm.DB
	cache sharedcache.Cache

	inTransaction bool
	afterCommit   []func(context.Context)

	accountRepo          repos.AccountRepository
	accountAggregateRepo repos.AccountAggregateRepository
}

func NewRepoImpl(db *gorm.DB, cache sharedcache.Cache) repos.Repos {
	return newRepoImplWithDB(db, cache, false)
}

func newRepoImplWithDB(db *gorm.DB, cache sharedcache.Cache, inTransaction bool) *repoImpl {
	r := &repoImpl{
		db:            db,
		cache:         cache,
		inTransaction: inTransaction,
	}
	r.accountRepo = NewAccountRepoImpl(db, cache, !inTransaction, r.runAfterCommit)
	r.accountAggregateRepo = NewAccountAggregateRepoImpl(db, cache, r.runAfterCommit)
	return r
}

func (r *repoImpl) AccountRepository() repos.AccountRepository {
	return r.accountRepo
}

func (r *repoImpl) AccountAggregateRepository() repos.AccountAggregateRepository {
	return r.accountAggregateRepo
}

func (r *repoImpl) WithTransaction(ctx context.Context, fn func(repos.Repos) error) (err error) {
	log := logging.FromContext(ctx).Named("StartTransaction")
	tx := r.db.WithContext(ctx).Begin()
	if beginErr := tx.Error; beginErr != nil {
		log.Errorw("Failed to begin transaction", zap.Error(beginErr))
		return beginErr
	}
	tr := newRepoImplWithDB(tx, r.cache, true)

	defer func() {
		if rec := recover(); rec != nil {
			_ = tx.Rollback().Error
			log.Errorw("Panic -> rollback", zap.Any("panic", rec))
			panic(rec)
		}
		if err != nil {
			_ = tx.Rollback().Error
			log.Errorw("Transaction rollback", zap.Error(err))
			return
		}
		if commitErr := tx.Commit().Error; commitErr != nil {
			log.Errorw("Commit failed", zap.Error(commitErr))
			err = commitErr
		} else {
			tr.flushAfterCommit(ctx)
			log.Infow("Transaction committed")
		}
	}()

	err = fn(tr)

	return err
}

func (r *repoImpl) runAfterCommit(ctx context.Context, fn func(context.Context)) {
	if fn == nil {
		return
	}
	if !r.inTransaction {
		fn(ctx)
		return
	}
	r.afterCommit = append(r.afterCommit, fn)
}

func (r *repoImpl) flushAfterCommit(ctx context.Context) {
	for _, fn := range r.afterCommit {
		fn(ctx)
	}
	r.afterCommit = nil
}
