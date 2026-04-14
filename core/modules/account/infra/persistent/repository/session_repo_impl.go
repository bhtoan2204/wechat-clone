package repos

import (
	"context"
	"fmt"

	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/account/domain/entity"
	accountrepos "go-socket/core/modules/account/domain/repos"
	accountcache "go-socket/core/modules/account/infra/cache"
	"go-socket/core/modules/account/infra/persistent/models"
	sharedcache "go-socket/core/shared/infra/cache"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type sessionRepoImpl struct {
	db            *gorm.DB
	sessionCache  accountcache.SessionCache
	readFromCache bool
	afterCommit   afterCommitRegistrar
}

func NewSessionRepoImpl(
	db *gorm.DB,
	cache sharedcache.Cache,
	readFromCache bool,
	afterCommit afterCommitRegistrar,
) accountrepos.SessionRepository {
	if afterCommit == nil {
		afterCommit = func(ctx context.Context, fn func(context.Context)) {
			if fn != nil {
				fn(ctx)
			}
		}
	}

	return &sessionRepoImpl{
		db:            db,
		sessionCache:  accountcache.NewSessionCache(cache),
		readFromCache: readFromCache,
		afterCommit:   afterCommit,
	}
}

func (r *sessionRepoImpl) Load(ctx context.Context, sessionID string) (*aggregate.SessionAggregate, error) {
	if r.readFromCache {
		if cached, ok, err := r.sessionCache.Get(ctx, sessionID); err == nil && ok {
			return r.toAggregate(cached)
		}
	}

	var model models.SessionModel
	if err := r.db.WithContext(ctx).
		Where("id = ?", sessionID).
		First(&model).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	session, err := r.toEntity(&model)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if r.readFromCache {
		r.afterCommit(ctx, func(hookCtx context.Context) {
			_ = r.sessionCache.Set(hookCtx, session)
		})
	}
	return r.toAggregate(session)
}

func (r *sessionRepoImpl) Save(ctx context.Context, session *aggregate.SessionAggregate) error {
	if session == nil {
		return stackErr.Error(fmt.Errorf("session is nil"))
	}

	snapshot, err := session.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}
	if err := r.db.WithContext(ctx).Save(r.toModel(snapshot)).Error; err != nil {
		return stackErr.Error(err)
	}
	r.syncCacheAfterCommit(ctx, snapshot)
	return nil
}

func (r *sessionRepoImpl) ListByAccountID(ctx context.Context, accountID string) ([]*aggregate.SessionAggregate, error) {
	var modelsList []models.SessionModel
	if err := r.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Find(&modelsList).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	result := make([]*aggregate.SessionAggregate, 0, len(modelsList))
	for _, model := range modelsList {
		session, err := r.toEntity(&model)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		agg, err := r.toAggregate(session)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		result = append(result, agg)
	}
	return result, nil
}

func (r *sessionRepoImpl) toEntity(model *models.SessionModel) (*entity.Session, error) {
	if model == nil {
		return nil, stackErr.Error(fmt.Errorf("session model is nil"))
	}

	return &entity.Session{
		ID:               model.ID,
		AccountID:        model.AccountID,
		DeviceID:         model.DeviceID,
		RefreshTokenHash: model.RefreshTokenHash,
		Status:           entity.SessionStatus(model.Status),
		IPAddress:        cloneString(model.IPAddress),
		UserAgent:        cloneString(model.UserAgent),
		LastActivityAt:   cloneTime(model.LastActivityAt),
		ExpiresAt:        model.ExpiresAt,
		RevokedAt:        cloneTime(model.RevokedAt),
		RevokedReason:    cloneString(model.RevokedReason),
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}, nil
}

func (r *sessionRepoImpl) toAggregate(session *entity.Session) (*aggregate.SessionAggregate, error) {
	agg, err := aggregate.NewSessionAggregate(session.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.Restore(session); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *sessionRepoImpl) toModel(session *entity.Session) *models.SessionModel {
	return &models.SessionModel{
		ID:               session.ID,
		AccountID:        session.AccountID,
		DeviceID:         session.DeviceID,
		RefreshTokenHash: session.RefreshTokenHash,
		Status:           string(session.Status),
		IPAddress:        cloneString(session.IPAddress),
		UserAgent:        cloneString(session.UserAgent),
		LastActivityAt:   cloneTime(session.LastActivityAt),
		ExpiresAt:        session.ExpiresAt,
		RevokedAt:        cloneTime(session.RevokedAt),
		RevokedReason:    cloneString(session.RevokedReason),
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
	}
}

func (r *sessionRepoImpl) syncCacheAfterCommit(ctx context.Context, session *entity.Session) {
	if session == nil {
		return
	}

	sessionClone := *session
	sessionClone.IPAddress = cloneString(session.IPAddress)
	sessionClone.UserAgent = cloneString(session.UserAgent)
	sessionClone.LastActivityAt = cloneTime(session.LastActivityAt)
	sessionClone.RevokedAt = cloneTime(session.RevokedAt)
	sessionClone.RevokedReason = cloneString(session.RevokedReason)

	r.afterCommit(ctx, func(hookCtx context.Context) {
		_ = r.sessionCache.Set(hookCtx, &sessionClone)
	})
}
