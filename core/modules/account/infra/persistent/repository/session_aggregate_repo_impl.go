package repos

import (
	"context"
	"fmt"

	"wechat-clone/core/modules/account/domain/aggregate"
	"wechat-clone/core/modules/account/domain/entity"
	accountrepos "wechat-clone/core/modules/account/domain/repos"
	accountcache "wechat-clone/core/modules/account/infra/cache"
	"wechat-clone/core/modules/account/infra/persistent/models"
	sharedcache "wechat-clone/core/shared/infra/cache"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type sessionAggregateRepoImpl struct {
	db            *gorm.DB
	sessionCache  accountcache.SessionCache
	readFromCache bool
	afterCommit   afterCommitRegistrar
}

func NewSessionAggregateRepoImpl(
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

	return &sessionAggregateRepoImpl{
		db:            db,
		sessionCache:  accountcache.NewSessionCache(cache),
		readFromCache: readFromCache,
		afterCommit:   afterCommit,
	}
}

func (r *sessionAggregateRepoImpl) Load(ctx context.Context, sessionID string) (*aggregate.SessionAggregate, error) {
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

func (r *sessionAggregateRepoImpl) Save(ctx context.Context, session *aggregate.SessionAggregate) error {
	if session == nil {
		return stackErr.Error(fmt.Errorf("session is nil"))
	}

	snapshot, err := session.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"account_id",
				"device_id",
				"refresh_token_hash",
				"status",
				"ip_address",
				"user_agent",
				"last_activity_at",
				"expires_at",
				"revoked_at",
				"revoked_reason",
				"updated_at",
			}),
		}).
		Create(r.toModel(snapshot)).Error; err != nil {
		return stackErr.Error(err)
	}
	r.syncCacheAfterCommit(ctx, snapshot)
	return nil
}

func (r *sessionAggregateRepoImpl) ListByAccountID(ctx context.Context, accountID string) ([]*aggregate.SessionAggregate, error) {
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

func (r *sessionAggregateRepoImpl) toEntity(model *models.SessionModel) (*entity.Session, error) {
	if model == nil {
		return nil, stackErr.Error(fmt.Errorf("session model is nil"))
	}

	return &entity.Session{
		ID:               model.ID,
		AccountID:        model.AccountID,
		DeviceID:         model.DeviceID,
		RefreshTokenHash: model.RefreshTokenHash,
		Status:           entity.SessionStatus(model.Status),
		IPAddress:        utils.ClonePtr(model.IPAddress),
		UserAgent:        utils.ClonePtr(model.UserAgent),
		LastActivityAt:   utils.ClonePtr(model.LastActivityAt),
		ExpiresAt:        model.ExpiresAt,
		RevokedAt:        utils.ClonePtr(model.RevokedAt),
		RevokedReason:    utils.ClonePtr(model.RevokedReason),
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}, nil
}

func (r *sessionAggregateRepoImpl) toAggregate(session *entity.Session) (*aggregate.SessionAggregate, error) {
	agg, err := aggregate.NewSessionAggregate(session.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := agg.Restore(session); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *sessionAggregateRepoImpl) toModel(session *entity.Session) *models.SessionModel {
	return &models.SessionModel{
		ID:               session.ID,
		AccountID:        session.AccountID,
		DeviceID:         session.DeviceID,
		RefreshTokenHash: session.RefreshTokenHash,
		Status:           session.Status.String(),
		IPAddress:        utils.ClonePtr(session.IPAddress),
		UserAgent:        utils.ClonePtr(session.UserAgent),
		LastActivityAt:   utils.ClonePtr(session.LastActivityAt),
		ExpiresAt:        session.ExpiresAt,
		RevokedAt:        utils.ClonePtr(session.RevokedAt),
		RevokedReason:    utils.ClonePtr(session.RevokedReason),
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
	}
}

func (r *sessionAggregateRepoImpl) syncCacheAfterCommit(ctx context.Context, session *entity.Session) {
	if session == nil {
		return
	}

	sessionClone := *session
	sessionClone.IPAddress = utils.ClonePtr(session.IPAddress)
	sessionClone.UserAgent = utils.ClonePtr(session.UserAgent)
	sessionClone.LastActivityAt = utils.ClonePtr(session.LastActivityAt)
	sessionClone.RevokedAt = utils.ClonePtr(session.RevokedAt)
	sessionClone.RevokedReason = utils.ClonePtr(session.RevokedReason)

	r.afterCommit(ctx, func(hookCtx context.Context) {
		_ = r.sessionCache.Set(hookCtx, &sessionClone)
	})
}
