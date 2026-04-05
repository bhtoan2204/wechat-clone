package repository

import (
	"context"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"

	"gorm.io/gorm"
)

type roomMemberImpl struct {
	db *gorm.DB
}

func NewRoomMemberImpl(db *gorm.DB) repos.RoomMemberRepository {
	return &roomMemberImpl{db: db}
}

func (r *roomMemberImpl) CreateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error {
	m := r.toModel(roomMember)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

func (r *roomMemberImpl) DeleteRoomMember(ctx context.Context, roomID, accountID string) error {
	return r.db.WithContext(ctx).Where("room_id = ? AND account_id = ?", roomID, accountID).Delete(&models.RoomMemberModel{}).Error
}

func (r *roomMemberImpl) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error) {
	var m models.RoomMemberModel
	if err := r.db.WithContext(ctx).Where("room_id = ? AND account_id = ?", roomID, accountID).First(&m).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *roomMemberImpl) toModel(e *entity.RoomMemberEntity) *models.RoomMemberModel {
	return &models.RoomMemberModel{
		ID:              e.ID,
		RoomID:          e.RoomID,
		AccountID:       e.AccountID,
		Role:            e.Role,
		LastDeliveredAt: e.LastDeliveredAt,
		LastReadAt:      e.LastReadAt,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func (r *roomMemberImpl) toEntity(m *models.RoomMemberModel) *entity.RoomMemberEntity {
	return &entity.RoomMemberEntity{
		ID:              m.ID,
		RoomID:          m.RoomID,
		AccountID:       m.AccountID,
		Role:            m.Role,
		LastDeliveredAt: m.LastDeliveredAt,
		LastReadAt:      m.LastReadAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}
