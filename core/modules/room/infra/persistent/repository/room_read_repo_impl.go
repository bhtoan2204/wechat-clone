package repository

import (
	"context"
	"strings"
	"time"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"
	"go-socket/core/shared/utils"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type roomReadRepoImpl struct {
	db *gorm.DB
}

func NewRoomReadRepoImpl(db *gorm.DB) repos.RoomReadRepository {
	return &roomReadRepoImpl{db: db}
}

func (r *roomReadRepoImpl) UpsertRoom(ctx context.Context, room *entity.Room) error {
	modelRoom := r.toModel(room)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(modelRoom).Error
}

func (r *roomReadRepoImpl) UpdateRoom(ctx context.Context, room *entity.Room) error {
	updates := map[string]interface{}{
		"name":        room.Name,
		"description": room.Description,
		"room_type":   room.RoomType,
		"owner_id":    room.OwnerID,
		"direct_key":  roomNullableString(room.DirectKey),
		"updated_at":  room.UpdatedAt,
	}
	if room.PinnedMessageID != "" {
		updates["pinned_message_id"] = room.PinnedMessageID
	} else {
		updates["pinned_message_id"] = nil
	}

	return r.db.WithContext(ctx).Model(&models.RoomReadModel{}).Where("id = ?", room.ID).Updates(updates).Error
}

func (r *roomReadRepoImpl) DeleteRoom(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.RoomReadModel{}, "id = ?", id).Error
}

func (r *roomReadRepoImpl) ListRooms(ctx context.Context, options utils.QueryOptions) ([]*entity.Room, error) {
	var rooms []*models.RoomReadModel
	tx := r.db.WithContext(ctx).Model(&models.RoomReadModel{})
	for _, c := range options.Conditions {
		tx = tx.Where(c.BuildCondition(), c.Value)
	}
	if options.Limit != nil {
		tx = tx.Limit(*options.Limit)
	}
	if options.Offset != nil {
		tx = tx.Offset(*options.Offset)
	}
	if options.OrderBy != "" && options.OrderDirection != "" {
		tx = tx.Order(options.OrderBy + " " + options.OrderDirection)
	}
	if err := tx.Find(&rooms).Error; err != nil {
		return nil, err
	}

	return lo.Map(rooms, func(room *models.RoomReadModel, _ int) *entity.Room {
		return r.toEntity(room)
	}), nil
}

func (r *roomReadRepoImpl) ListRoomsByAccount(ctx context.Context, accountID string, options utils.QueryOptions) ([]*entity.Room, error) {
	var rooms []*models.RoomReadModel
	tx := r.db.WithContext(ctx).
		Table("room_read_models rr").
		Select("rr.*").
		Joins("JOIN room_member_read_models rm ON rm.room_id = rr.id").
		Where("rm.account_id = ?", strings.TrimSpace(accountID))
	for _, c := range options.Conditions {
		tx = tx.Where(c.BuildCondition(), c.Value)
	}
	if options.Limit != nil {
		tx = tx.Limit(*options.Limit)
	}
	if options.Offset != nil {
		tx = tx.Offset(*options.Offset)
	}
	if options.OrderBy != "" && options.OrderDirection != "" {
		tx = tx.Order(options.OrderBy + " " + options.OrderDirection)
	} else {
		tx = tx.Order("rr.updated_at DESC")
	}
	if err := tx.Find(&rooms).Error; err != nil {
		return nil, err
	}

	return lo.Map(rooms, func(room *models.RoomReadModel, _ int) *entity.Room {
		return r.toEntity(room)
	}), nil
}

func (r *roomReadRepoImpl) GetRoomByID(ctx context.Context, id string) (*entity.Room, error) {
	var room models.RoomReadModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&room).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&room), nil
}

func (r *roomReadRepoImpl) UpdateRoomStats(ctx context.Context, roomID string, memberCount int, lastMessage *entity.MessageEntity, updatedAt time.Time) error {
	updates := map[string]interface{}{
		"member_count": memberCount,
		"updated_at":   updatedAt,
	}
	if lastMessage != nil {
		updates["last_message_id"] = lastMessage.ID
		updates["last_message_at"] = lastMessage.CreatedAt
		updates["last_message_content"] = lastMessage.Message
		updates["last_message_sender_id"] = lastMessage.SenderID
	} else {
		updates["last_message_id"] = nil
		updates["last_message_at"] = nil
		updates["last_message_content"] = nil
		updates["last_message_sender_id"] = nil
	}

	return r.db.WithContext(ctx).Model(&models.RoomReadModel{}).Where("id = ?", roomID).Updates(updates).Error
}

func (r *roomReadRepoImpl) UpdatePinnedMessage(ctx context.Context, roomID, pinnedMessageID string, updatedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.RoomReadModel{}).Where("id = ?", roomID).Updates(map[string]interface{}{
		"pinned_message_id": pinnedMessageID,
		"updated_at":        updatedAt,
	}).Error
}

func (r *roomReadRepoImpl) toEntity(m *models.RoomReadModel) *entity.Room {
	return &entity.Room{
		ID:              m.ID,
		Name:            m.Name,
		Description:     m.Description,
		RoomType:        m.RoomType,
		OwnerID:         m.OwnerID,
		DirectKey:       roomDerefString(m.DirectKey),
		PinnedMessageID: roomDerefString(m.PinnedMessageID),
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func (r *roomReadRepoImpl) toModel(e *entity.Room) *models.RoomReadModel {
	return &models.RoomReadModel{
		ID:              e.ID,
		Name:            e.Name,
		Description:     e.Description,
		RoomType:        e.RoomType,
		OwnerID:         e.OwnerID,
		DirectKey:       roomNullableString(e.DirectKey),
		PinnedMessageID: roomNullableString(e.PinnedMessageID),
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}
