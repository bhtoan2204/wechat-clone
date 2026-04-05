package repository

import (
	"context"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type roomMemberReadRepoImpl struct {
	db *gorm.DB
}

func NewRoomMemberReadRepoImpl(db *gorm.DB) repos.RoomMemberReadRepository {
	return &roomMemberReadRepoImpl{db: db}
}

func (r *roomMemberReadRepoImpl) UpsertRoomMember(ctx context.Context, roomMember *entity.RoomMemberEntity) error {
	modelMember := r.toModel(roomMember)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(modelMember).Error
}

func (r *roomMemberReadRepoImpl) DeleteRoomMember(ctx context.Context, roomID, accountID string) error {
	return r.db.WithContext(ctx).Delete(&models.RoomMemberReadModel{}, "room_id = ? AND account_id = ?", roomID, accountID).Error
}

func (r *roomMemberReadRepoImpl) ListRoomMembers(ctx context.Context, roomID string) ([]*entity.RoomMemberEntity, error) {
	var members []*models.RoomMemberReadModel
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Order("created_at ASC").Find(&members).Error; err != nil {
		return nil, err
	}
	return lo.Map(members, func(member *models.RoomMemberReadModel, _ int) *entity.RoomMemberEntity {
		return r.toEntity(member)
	}), nil
}

func (r *roomMemberReadRepoImpl) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error) {
	var member models.RoomMemberReadModel
	if err := r.db.WithContext(ctx).Where("room_id = ? AND account_id = ?", roomID, accountID).First(&member).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&member), nil
}

func (r *roomMemberReadRepoImpl) toModel(e *entity.RoomMemberEntity) *models.RoomMemberReadModel {
	return &models.RoomMemberReadModel{
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

func (r *roomMemberReadRepoImpl) toEntity(m *models.RoomMemberReadModel) *entity.RoomMemberEntity {
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
