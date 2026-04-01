package repository

import (
	"context"
	"fmt"
	"go-socket/core/modules/payment/domain/entity"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/model"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type paymentAccountProjectionRepoImpl struct {
	db *gorm.DB
}

func NewPaymentAccountProjectionRepoImpl(db *gorm.DB) paymentrepos.PaymentAccountProjectionRepository {
	return &paymentAccountProjectionRepoImpl{db: db}
}

func (p *paymentAccountProjectionRepoImpl) GetAccountProjectionByAccountID(ctx context.Context, accountID string) (*entity.PaymentAccount, error) {
	var accountProjection model.PaymentAccountProjectionModel
	if err := p.db.WithContext(ctx).Where("account_id = ?", accountID).First(&accountProjection).Error; err != nil {
		return nil, stackerr.Error(err)
	}
	return &entity.PaymentAccount{
		ID:        accountProjection.ID,
		AccountID: accountProjection.AccountID,
		Email:     accountProjection.Email,
		CreatedAt: accountProjection.CreatedAt,
	}, nil
}

func (p *paymentAccountProjectionRepoImpl) CreateAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error {
	if accountProjection == nil {
		return fmt.Errorf("account projection is nil")
	}
	modelProjection, err := toProjectionModel(accountProjection)
	if err != nil {
		return err
	}
	return p.db.WithContext(ctx).Create(&modelProjection).Error
}

func (p *paymentAccountProjectionRepoImpl) UpdateAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error {
	if accountProjection == nil {
		return fmt.Errorf("account projection is nil")
	}
	accountID := accountProjection.AccountID
	if accountID == "" {
		accountID = accountProjection.ID
	}
	if accountID == "" {
		return fmt.Errorf("account id is empty")
	}
	updates := map[string]interface{}{
		"email": accountProjection.Email,
	}
	return p.db.WithContext(ctx).
		Model(&model.PaymentAccountProjectionModel{}).
		Where("account_id = ?", accountID).
		Updates(updates).Error
}

func (p *paymentAccountProjectionRepoImpl) DeleteAccountProjection(ctx context.Context, accountID string) error {
	return p.db.WithContext(ctx).Delete(&model.PaymentAccountProjectionModel{}, "account_id = ?", accountID).Error
}

func toProjectionModel(accountProjection *entity.PaymentAccount) (model.PaymentAccountProjectionModel, error) {
	accountID := accountProjection.AccountID
	if accountID == "" {
		accountID = accountProjection.ID
	}
	if accountID == "" {
		return model.PaymentAccountProjectionModel{}, fmt.Errorf("account id is empty")
	}
	id := accountProjection.ID
	if id == "" {
		id = accountID
	}
	return model.PaymentAccountProjectionModel{
		ID:        id,
		AccountID: accountID,
		Email:     accountProjection.Email,
		CreatedAt: accountProjection.CreatedAt,
	}, nil
}

func (p *paymentAccountProjectionRepoImpl) UpsertAccountProjection(ctx context.Context, accountProjection *entity.PaymentAccount) error {
	if accountProjection == nil {
		return fmt.Errorf("account projection is nil")
	}

	modelProjection, err := toProjectionModel(accountProjection)
	if err != nil {
		return err
	}

	return p.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_id"}},
		UpdateAll: true,
	}).Create(&modelProjection).Error
}
