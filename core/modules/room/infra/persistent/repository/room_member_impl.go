package repository

import (
	"context"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

type roomMemberImpl struct {
	db *gorm.DB
}

func NewRoomMemberImpl(db *gorm.DB) *roomMemberImpl {
	return &roomMemberImpl{db: db}
}

func (r *roomMemberImpl) CreateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error {
	m := r.toModel(roomMember)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *roomMemberImpl) DeleteRoomMember(ctx context.Context, roomID, accountID string) error {
	return r.db.WithContext(ctx).Where("room_id = ? AND account_id = ?", roomID, accountID).Delete(&models.RoomMemberModel{}).Error
}

func (r *roomMemberImpl) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error) {
	var m models.RoomMemberModel
	if err := r.db.WithContext(ctx).Where("room_id = ? AND account_id = ?", roomID, accountID).First(&m).Error; err != nil {
		return nil, stackErr.Error(err)
	}
	return r.toEntity(&m), nil
}

func (r *roomMemberImpl) ListRoomMembers(ctx context.Context, roomID string) ([]*entity.RoomMemberEntity, error) {
	var members []*models.RoomMemberModel
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Order("created_at ASC").Find(&members).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	return lo.Map(members, func(member *models.RoomMemberModel, _ int) *entity.RoomMemberEntity {
		return r.toEntity(member)
	}), nil
}

func (r *roomMemberImpl) UpdateRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error {
	return stackErr.Error(r.db.WithContext(ctx).
		Model(&models.RoomMemberModel{}).
		Where("id = ?", roomMember.ID).
		Updates(map[string]interface{}{
			"role":              roomMember.Role,
			"last_delivered_at": roomMember.LastDeliveredAt,
			"last_read_at":      roomMember.LastReadAt,
			"updated_at":        roomMember.UpdatedAt,
		}).Error)
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
