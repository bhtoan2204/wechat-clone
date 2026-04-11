package repository

import (
	"context"
	"strings"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"
	"go-socket/core/shared/pkg/stackErr"

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
		return nil, stackErr.Error(err)
	}
	return lo.Map(members, func(member *models.RoomMemberReadModel, _ int) *entity.RoomMemberEntity {
		return r.toEntity(member)
	}), nil
}

func (r *roomMemberReadRepoImpl) GetRoomMemberByAccount(ctx context.Context, roomID, accountID string) (*entity.RoomMemberEntity, error) {
	var member models.RoomMemberReadModel
	if err := r.db.WithContext(ctx).Where("room_id = ? AND account_id = ?", roomID, accountID).First(&member).Error; err != nil {
		return nil, stackErr.Error(err)
	}
	return r.toEntity(&member), nil
}

func (r *roomMemberReadRepoImpl) SearchMentionCandidates(ctx context.Context, roomID, keyword, excludeAccountID string, limit int) ([]*entity.MentionCandidate, error) {
	type mentionCandidateRow struct {
		AccountID       string
		DisplayName     string
		Username        string
		AvatarObjectKey string
	}

	normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
	rows := make([]mentionCandidateRow, 0)
	tx := r.db.WithContext(ctx).
		Table("room_member_read_models rm").
		Select(`
			rm.account_id AS account_id,
			COALESCE(rap.display_name, '') AS display_name,
			COALESCE(rap.username, '') AS username,
			COALESCE(rap.avatar_object_key, '') AS avatar_object_key
		`).
		Joins("LEFT JOIN room_account_projections rap ON rap.account_id = rm.account_id").
		Where("rm.room_id = ?", strings.TrimSpace(roomID))

	if exclude := strings.TrimSpace(excludeAccountID); exclude != "" {
		tx = tx.Where("rm.account_id <> ?", exclude)
	}

	if normalizedKeyword != "" {
		like := "%" + normalizedKeyword + "%"
		tx = tx.Where(
			"(LOWER(COALESCE(rap.display_name, '')) LIKE ? OR LOWER(COALESCE(rap.username, '')) LIKE ? OR LOWER(rm.account_id) LIKE ?)",
			like,
			like,
			like,
		)
	}

	if err := tx.
		Order("LOWER(COALESCE(rap.display_name, rm.account_id)) ASC").
		Order("LOWER(COALESCE(rap.username, '')) ASC").
		Order("rm.created_at ASC").
		Limit(limitOrDefault(limit, 20, 50)).
		Find(&rows).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	return lo.Map(rows, func(item mentionCandidateRow, _ int) *entity.MentionCandidate {
		return &entity.MentionCandidate{
			AccountID:       item.AccountID,
			DisplayName:     item.DisplayName,
			Username:        item.Username,
			AvatarObjectKey: item.AvatarObjectKey,
		}
	}), nil
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
