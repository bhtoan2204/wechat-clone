package repository

import (
	"context"
	"sort"
	"strings"
	"time"

	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type messageReadRepoImpl struct {
	db *gorm.DB
}

func NewMessageReadRepoImpl(db *gorm.DB) repos.MessageReadRepository {
	return &messageReadRepoImpl{db: db}
}

func (r *messageReadRepoImpl) UpsertMessage(ctx context.Context, message *entity.MessageEntity) error {
	modelMessage := r.toModel(message)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(modelMessage).Error
}

func (r *messageReadRepoImpl) GetMessageByID(ctx context.Context, id string) (*entity.MessageEntity, error) {
	var message models.MessageReadModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&message).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&message), nil
}

func (r *messageReadRepoImpl) GetLastMessage(ctx context.Context, roomID string) (*entity.MessageEntity, error) {
	var message models.MessageReadModel
	if err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		First(&message).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&message), nil
}

func (r *messageReadRepoImpl) ListMessages(ctx context.Context, accountID, roomID string, options repos.MessageListOptions) ([]*entity.MessageEntity, error) {
	tx := r.db.WithContext(ctx).Model(&models.MessageReadModel{}).
		Where("room_id = ?", roomID).
		Where("id NOT IN (SELECT message_id FROM message_deletion_read_models WHERE account_id = ?)", strings.TrimSpace(accountID))

	if beforeID := strings.TrimSpace(options.BeforeID); beforeID != "" {
		var before models.MessageReadModel
		if err := r.db.WithContext(ctx).Where("id = ?", beforeID).First(&before).Error; err == nil {
			tx = tx.Where("created_at < ?", before.CreatedAt)
		}
	}
	if options.BeforeAt != nil {
		tx = tx.Where("created_at < ?", *options.BeforeAt)
	}

	order := "created_at DESC"
	if options.Ascending {
		order = "created_at ASC"
	}

	var messages []*models.MessageReadModel
	if err := tx.Order(order).Limit(limitOrDefault(options.Limit, 50, 200)).Find(&messages).Error; err != nil {
		return nil, err
	}
	if !options.Ascending {
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		})
	}

	return lo.Map(messages, func(message *models.MessageReadModel, _ int) *entity.MessageEntity {
		return r.toEntity(message)
	}), nil
}

func (r *messageReadRepoImpl) UpsertMessageReceipt(ctx context.Context, messageID, accountID, status string, deliveredAt, seenAt *time.Time, createdAt, updatedAt time.Time) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}, {Name: "account_id"}},
		UpdateAll: true,
	}).Create(&models.MessageReceiptReadModel{
		ID:          messageID + ":" + accountID,
		MessageID:   messageID,
		AccountID:   accountID,
		Status:      status,
		DeliveredAt: deliveredAt,
		SeenAt:      seenAt,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}).Error
}

func (r *messageReadRepoImpl) GetMessageReceipt(ctx context.Context, messageID, accountID string) (string, *time.Time, *time.Time, error) {
	var receipt models.MessageReceiptReadModel
	if err := r.db.WithContext(ctx).
		Where("message_id = ? AND account_id = ?", messageID, accountID).
		First(&receipt).Error; err != nil {
		return "", nil, nil, err
	}
	return receipt.Status, receipt.DeliveredAt, receipt.SeenAt, nil
}

func (r *messageReadRepoImpl) CountMessageReceiptsByStatus(ctx context.Context, messageID, status string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.MessageReceiptReadModel{}).
		Where("message_id = ? AND status = ?", messageID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *messageReadRepoImpl) UpsertMessageDeletion(ctx context.Context, messageID, accountID string, createdAt time.Time) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}, {Name: "account_id"}},
		UpdateAll: true,
	}).Create(&models.MessageDeletionReadModel{
		ID:        messageID + ":" + accountID,
		MessageID: messageID,
		AccountID: accountID,
		CreatedAt: createdAt,
	}).Error
}

func (r *messageReadRepoImpl) CountUnreadMessages(ctx context.Context, roomID, accountID string, lastReadAt *time.Time) (int64, error) {
	var count int64
	tx := r.db.WithContext(ctx).Model(&models.MessageReadModel{}).
		Where("room_id = ?", roomID).
		Where("sender_id <> ?", strings.TrimSpace(accountID)).
		Where("deleted_for_everyone_at IS NULL").
		Where("id NOT IN (SELECT message_id FROM message_deletion_read_models WHERE account_id = ?)", strings.TrimSpace(accountID))
	if lastReadAt != nil {
		tx = tx.Where("created_at > ?", *lastReadAt)
	}
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *messageReadRepoImpl) toModel(e *entity.MessageEntity) *models.MessageReadModel {
	return &models.MessageReadModel{
		ID:                     e.ID,
		RoomID:                 e.RoomID,
		SenderID:               e.SenderID,
		Message:                e.Message,
		MessageType:            e.MessageType,
		ReplyToMessageID:       nullableString(e.ReplyToMessageID),
		ForwardedFromMessageID: nullableString(e.ForwardedFromMessageID),
		FileName:               nullableString(e.FileName),
		FileSize:               int64Ptr(e.FileSize),
		MimeType:               nullableString(e.MimeType),
		ObjectKey:              nullableString(e.ObjectKey),
		EditedAt:               e.EditedAt,
		DeletedForEveryoneAt:   e.DeletedForEveryoneAt,
		CreatedAt:              e.CreatedAt,
	}
}

func (r *messageReadRepoImpl) toEntity(m *models.MessageReadModel) *entity.MessageEntity {
	return &entity.MessageEntity{
		ID:                     m.ID,
		RoomID:                 m.RoomID,
		SenderID:               m.SenderID,
		Message:                m.Message,
		MessageType:            m.MessageType,
		ReplyToMessageID:       derefString(m.ReplyToMessageID),
		ForwardedFromMessageID: derefString(m.ForwardedFromMessageID),
		FileName:               derefString(m.FileName),
		FileSize:               derefInt64(m.FileSize),
		MimeType:               derefString(m.MimeType),
		ObjectKey:              derefString(m.ObjectKey),
		EditedAt:               m.EditedAt,
		DeletedForEveryoneAt:   m.DeletedForEveryoneAt,
		CreatedAt:              m.CreatedAt,
	}
}
