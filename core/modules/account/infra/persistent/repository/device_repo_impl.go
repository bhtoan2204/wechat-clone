package repos

import (
	"context"
	"fmt"

	"go-socket/core/modules/account/domain/entity"
	accountrepos "go-socket/core/modules/account/domain/repos"
	"go-socket/core/modules/account/infra/persistent/models"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type deviceRepoImpl struct {
	db *gorm.DB
}

func NewDeviceRepoImpl(db *gorm.DB) accountrepos.DeviceRepository {
	return &deviceRepoImpl{db: db}
}

func (r *deviceRepoImpl) FindByAccountAndUID(ctx context.Context, accountID string, deviceUID string) (*entity.Device, error) {
	var model models.DeviceModel
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND device_uid = ?", accountID, deviceUID).
		First(&model).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	device, err := r.toEntity(&model)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return device, nil
}

func (r *deviceRepoImpl) GetByAccountAndID(ctx context.Context, accountID string, deviceID string) (*entity.Device, error) {
	var model models.DeviceModel
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND id = ?", accountID, deviceID).
		First(&model).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	device, err := r.toEntity(&model)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return device, nil
}

func (r *deviceRepoImpl) Save(ctx context.Context, device *entity.Device) error {
	if device == nil {
		return stackErr.Error(fmt.Errorf("device is nil"))
	}

	if err := r.db.WithContext(ctx).Save(r.toModel(device)).Error; err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *deviceRepoImpl) toEntity(model *models.DeviceModel) (*entity.Device, error) {
	if model == nil {
		return nil, stackErr.Error(fmt.Errorf("device model is nil"))
	}

	return &entity.Device{
		ID:            model.ID,
		AccountID:     model.AccountID,
		DeviceUID:     model.DeviceUID,
		DeviceName:    cloneString(model.DeviceName),
		DeviceType:    entity.DeviceType(model.DeviceType),
		OSName:        cloneString(model.OSName),
		OSVersion:     cloneString(model.OSVersion),
		AppVersion:    cloneString(model.AppVersion),
		UserAgent:     cloneString(model.UserAgent),
		LastIPAddress: cloneString(model.LastIPAddress),
		LastSeenAt:    cloneTime(model.LastSeenAt),
		IsTrusted:     model.IsTrusted == 1,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}, nil
}

func (r *deviceRepoImpl) toModel(device *entity.Device) *models.DeviceModel {
	isTrusted := int8(0)
	if device.IsTrusted {
		isTrusted = 1
	}

	return &models.DeviceModel{
		ID:            device.ID,
		AccountID:     device.AccountID,
		DeviceUID:     device.DeviceUID,
		DeviceName:    cloneString(device.DeviceName),
		DeviceType:    string(device.DeviceType),
		OSName:        cloneString(device.OSName),
		OSVersion:     cloneString(device.OSVersion),
		AppVersion:    cloneString(device.AppVersion),
		UserAgent:     cloneString(device.UserAgent),
		LastIPAddress: cloneString(device.LastIPAddress),
		LastSeenAt:    cloneTime(device.LastSeenAt),
		IsTrusted:     isTrusted,
		CreatedAt:     device.CreatedAt,
		UpdatedAt:     device.UpdatedAt,
	}
}
