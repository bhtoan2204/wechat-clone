package repos

import (
	"context"
	"strings"

	"wechat-clone/core/modules/account/domain/entity"
	accountrepos "wechat-clone/core/modules/account/domain/repos"
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
	db            *gorm.DB
	accountCache  accountcache.AccountCache
	readFromCache bool
	afterCommit   afterCommitRegistrar
}

func NewAccountRepoImpl(
	db *gorm.DB,
	sharedCache sharedcache.Cache,
	readFromCache bool,
	afterCommit afterCommitRegistrar,
) accountrepos.AccountRepository {
	if afterCommit == nil {
		afterCommit = func(ctx context.Context, fn func(context.Context)) {
			if fn != nil {
				fn(ctx)
			}
		}
	}

	return &accountRepoImpl{
		db:            db,
		accountCache:  accountcache.NewAccountCache(sharedCache),
		readFromCache: readFromCache,
		afterCommit:   afterCommit,
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

func (r *accountRepoImpl) IsEmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountModel{}).
		Where("email = ?", email).
		Count(&count).Error; err != nil {
		return false, stackErr.Error(err)
	}
	return count > 0, nil
}

func (r *accountRepoImpl) CreateAccount(ctx context.Context, account *entity.Account) error {
	m := r.toModel(account)

	if err := r.db.WithContext(ctx).
		Create(m).Error; err != nil {
		return stackErr.Error(err)
	}

	r.syncCacheAfterCommit(ctx, account)
	return nil
}

func (r *accountRepoImpl) UpdateAccount(ctx context.Context, account *entity.Account) error {
	m := r.toModel(account)

	if err := r.db.WithContext(ctx).
		Save(m).Error; err != nil {
		return stackErr.Error(err)
	}

	r.syncCacheAfterCommit(ctx, account)
	return nil
}

func (r *accountRepoImpl) DeleteAccount(ctx context.Context, id string) error {
	email, err := r.lookupAccountEmail(ctx, id)
	if err != nil {
		return stackErr.Error(err)
	}

	if err := r.db.WithContext(ctx).
		Delete(&models.AccountModel{}, "id = ?", id).Error; err != nil {
		return stackErr.Error(err)
	}

	r.afterCommit(ctx, func(hookCtx context.Context) {
		log := logging.FromContext(hookCtx).Named("DeleteAccountCache")
		if cacheErr := r.accountCache.Delete(hookCtx, id); cacheErr != nil {
			log.Errorw("Failed to delete account cache by id", zap.String("accountID", id))
		}
		if email != "" {
			if cacheErr := r.accountCache.DeleteByEmail(hookCtx, email); cacheErr != nil {
				log.Errorw("Failed to delete account cache by email", zap.String("email", email))
			}
		}
	})
	return nil
}

func (r *accountRepoImpl) ListAccountsByRoomID(ctx context.Context, roomID string) ([]*entity.Account, error) {
	var accounts []*models.AccountModel
	if err := r.db.WithContext(ctx).
		Model(&models.AccountModel{}).
		Select("accounts.*").
		Joins("JOIN room_members rm ON rm.account_id = accounts.id").
		Where("rm.room_id = ?", roomID).
		Find(&accounts).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	result := make([]*entity.Account, 0, len(accounts))

	for _, account := range accounts {
		e, err := r.toEntity(account)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		result = append(result, e)
	}

	return result, nil
}

func (r *accountRepoImpl) toEntity(m *models.AccountModel) (*entity.Account, error) {
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

func (r *accountRepoImpl) toModel(e *entity.Account) *models.AccountModel {
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

func (r *accountRepoImpl) syncCacheAfterCommit(ctx context.Context, account *entity.Account) {
	if account == nil {
		return
	}

	accountClone := *account
	r.afterCommit(ctx, func(hookCtx context.Context) {
		log := logging.FromContext(hookCtx).Named("SyncAccountCache")
		if cacheErr := r.accountCache.Set(hookCtx, &accountClone); cacheErr != nil {
			log.Errorw("Failed to update account cache by id", zap.String("accountID", accountClone.ID))
		}
		if cacheErr := r.accountCache.SetByEmail(hookCtx, &accountClone); cacheErr != nil {
			log.Errorw("Failed to update account cache by email", zap.String("email", accountClone.Email.Value()))
		}
	})
}

func (r *accountRepoImpl) lookupAccountEmail(ctx context.Context, id string) (string, error) {
	if r.readFromCache {
		if cached, ok, err := r.accountCache.Get(ctx, id); err == nil && ok {
			return cached.Email.Value(), nil
		}
	}

	var model models.AccountModel
	if err := r.db.WithContext(ctx).
		Select("email").
		Where("id = ?", id).
		First(&model).Error; err != nil {
		return "", stackErr.Error(err)
	}
	return model.Email, nil
}

func (r *accountRepoImpl) SearchUsers(ctx context.Context, q string, limit, offset int) ([]*entity.Account, int64, error) {
	q = strings.TrimSpace(strings.ToLower(q))
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
