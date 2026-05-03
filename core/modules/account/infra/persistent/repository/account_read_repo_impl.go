package repos

import (
	"context"
	"strings"

	accountprojection "wechat-clone/core/modules/account/application/projection"
	"wechat-clone/core/modules/account/domain/entity"
	valueobject "wechat-clone/core/modules/account/domain/value_object"
	accountcache "wechat-clone/core/modules/account/infra/cache"
	"wechat-clone/core/modules/account/infra/persistent/models"
	accounttypes "wechat-clone/core/modules/account/types"
	sharedcache "wechat-clone/core/shared/infra/cache"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type afterCommitRegistrar func(ctx context.Context, fn func(context.Context))

type accountRepoImpl struct {
	db               *gorm.DB
	accountCache     accountcache.AccountCache
	readFromCache    bool
	afterCommit      afterCommitRegistrar
	searchRepository accountprojection.SearchRepository
}

func NewAccountRepoImpl(
	db *gorm.DB,
	sharedCache sharedcache.Cache,
	readFromCache bool,
	afterCommit afterCommitRegistrar,
	searchRepository accountprojection.SearchRepository,
) accountprojection.AccountReadRepository {
	if afterCommit == nil {
		afterCommit = func(ctx context.Context, fn func(context.Context)) {
			if fn != nil {
				fn(ctx)
			}
		}
	}

	return &accountRepoImpl{
		db:               db,
		accountCache:     accountcache.NewAccountCache(sharedCache),
		readFromCache:    readFromCache,
		afterCommit:      afterCommit,
		searchRepository: searchRepository,
	}
}

func (r *accountRepoImpl) GetAccountByID(ctx context.Context, id string) (*entity.Account, error) {
	if r.readFromCache {
		if cached, ok, err := r.accountCache.Get(ctx, id); err == nil && ok {
			return cached, nil
		}
	}
	var m models.AccountModel

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&m).Error

	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountEntity, err := r.toEntity(&m)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if r.readFromCache {
		r.afterCommit(ctx, func(hookCtx context.Context) {
			log := logging.FromContext(hookCtx).Named("AccountCacheSetByID")
			if cacheErr := r.accountCache.Set(hookCtx, accountEntity); cacheErr != nil {
				log.Errorw("Failed to warm account cache", zap.String("accountID", accountEntity.ID))
			}
		})
	}

	return accountEntity, nil
}

func (r *accountRepoImpl) GetAccountByEmail(ctx context.Context, email string) (*entity.Account, error) {
	if r.readFromCache {
		if cached, ok, err := r.accountCache.GetByEmail(ctx, email); err == nil && ok {
			return cached, nil
		}
	}
	var m models.AccountModel
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&m).Error
	if err != nil {
		return nil, stackErr.Error(err)
	}
	accountEntity, err := r.toEntity(&m)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if r.readFromCache {
		r.afterCommit(ctx, func(hookCtx context.Context) {
			log := logging.FromContext(hookCtx).Named("AccountCacheSetByEmail")
			if cacheErr := r.accountCache.SetByEmail(hookCtx, accountEntity); cacheErr != nil {
				log.Errorw("Failed to warm account email cache", zap.String("email", accountEntity.Email.Value()))
			}
		})
	}
	return accountEntity, nil
}

func (r *accountRepoImpl) toEntity(m *models.AccountModel) (*entity.Account, error) {
	return projectionModelToAccount(m)
}

func (r *accountRepoImpl) SearchUsers(ctx context.Context, q string, limit, offset int) ([]*entity.Account, int64, error) {
	q = strings.TrimSpace(q)
	if r.searchRepository != nil {
		accounts, total, err := r.searchRepository.SearchUsers(ctx, q, limit, offset)
		if err != nil {
			return nil, 0, stackErr.Error(err)
		}
		return accounts, total, nil
	}

	q = strings.ToLower(q)
	if q == "" {
		return []*entity.Account{}, 0, nil
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	prefixQuery := q + "%"
	containsQuery := "%" + q + "%"

	baseQuery := r.db.WithContext(ctx).
		Model(&models.AccountModel{}).
		Where("status = ?", accounttypes.AccountStatusActive.String()).
		Where(r.db.WithContext(ctx).Where(
			"USERNAME_NORM LIKE ?",
			prefixQuery,
		).Or(
			"DISPLAY_NAME_NORM LIKE ?",
			containsQuery,
		).Or(
			"EMAIL_NORM LIKE ?",
			prefixQuery,
		))

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, stackErr.Error(err)
	}

	if total == 0 {
		return []*entity.Account{}, 0, nil
	}

	var accounts []*models.AccountModel
	if err := baseQuery.
		Order(gorm.Expr(`
			CASE
				WHEN USERNAME_NORM = ? THEN 1
				WHEN USERNAME_NORM LIKE ? THEN 2
				WHEN DISPLAY_NAME_NORM = ? THEN 3
				WHEN DISPLAY_NAME_NORM LIKE ? THEN 4
				WHEN EMAIL_NORM = ? THEN 5
				WHEN EMAIL_NORM LIKE ? THEN 6
				ELSE 7
			END
		`, q, prefixQuery, q, prefixQuery, q, prefixQuery)).
		Order("LAST_LOGIN_AT DESC NULLS LAST").
		Order("CREATED_AT DESC").
		Offset(offset).
		Limit(limit).
		Find(&accounts).Error; err != nil {
		return nil, 0, stackErr.Error(err)
	}

	result := make([]*entity.Account, 0, len(accounts))
	for _, account := range accounts {
		e, err := r.toEntity(account)
		if err != nil {
			return nil, 0, stackErr.Error(err)
		}
		result = append(result, e)
	}

	return result, total, nil
}

func projectionModelToAccount(m *models.AccountModel) (*entity.Account, error) {
	email, err := valueobject.NewEmail(m.Email)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	passwordHash, err := valueobject.NewHashedPassword(m.Password)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	status, err := accounttypes.ParseAccountStatus(m.Status)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &entity.Account{
		ID:                m.ID,
		Email:             email,
		PasswordHash:      passwordHash,
		DisplayName:       m.DisplayName,
		Username:          m.Username,
		AvatarObjectKey:   m.AvatarObjectKey,
		Status:            status,
		EmailVerifiedAt:   m.EmailVerifiedAt,
		LastLoginAt:       m.LastLoginAt,
		PasswordChangedAt: m.PasswordChangedAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		BannedReason:      m.BannedReason,
		BannedUntil:       m.BannedUntil,
	}, nil
}

func accountToProjectionModel(e *entity.Account) *models.AccountModel {
	return &models.AccountModel{
		ID:                e.ID,
		Email:             e.Email.Value(),
		Password:          e.PasswordHash.Value(),
		DisplayName:       e.DisplayName,
		Username:          e.Username,
		AvatarObjectKey:   e.AvatarObjectKey,
		Status:            e.Status.String(),
		EmailVerifiedAt:   e.EmailVerifiedAt,
		LastLoginAt:       e.LastLoginAt,
		PasswordChangedAt: e.PasswordChangedAt,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
		BannedReason:      e.BannedReason,
		BannedUntil:       e.BannedUntil,
	}
}
