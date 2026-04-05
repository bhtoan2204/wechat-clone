package repository

import (
	"context"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"

	"gorm.io/gorm"
)

type messageRepoImpl struct {
	db *gorm.DB
}

func NewMessageRepoImpl(db *gorm.DB) repos.MessageRepository {
	return &messageRepoImpl{db: db}
}

func (r *messageRepoImpl) CreateMessage(ctx context.Context, message *entity.MessageEntity) error {
	m := r.toModel(message)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

func (r *messageRepoImpl) UpdateMessage(ctx context.Context, message *entity.MessageEntity) error {
	m := r.toModel(message)
	return r.db.WithContext(ctx).Model(&models.MessageModel{}).Where("id = ?", message.ID).Updates(map[string]interface{}{
		"room_id":                   m.RoomID,
		"sender_id":                 m.SenderID,
		"message":                   m.Message,
		"message_type":              m.MessageType,
		"reply_to_message_id":       m.ReplyToMessageID,
		"forwarded_from_message_id": m.ForwardedFromMessageID,
		"file_name":                 m.FileName,
		"file_size":                 m.FileSize,
		"mime_type":                 m.MimeType,
		"object_key":                m.ObjectKey,
		"edited_at":                 m.EditedAt,
		"deleted_for_everyone_at":   m.DeletedForEveryoneAt,
		"created_at":                m.CreatedAt,
	}).Error
}

func (r *messageRepoImpl) GetMessageByID(ctx context.Context, id string) (*entity.MessageEntity, error) {
	var m models.MessageModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return r.toEntity(&m), nil
}

func (r *messageRepoImpl) toModel(e *entity.MessageEntity) *models.MessageModel {
	return &models.MessageModel{
		ID:                     e.ID,
		RoomID:                 e.RoomID,
		SenderID:               e.SenderID,
		Message:                e.Message,
		MessageType:            e.MessageType,
		ReplyToMessageID:       nullableString(e.ReplyToMessageID),
		ForwardedFromMessageID: nullableString(e.ForwardedFromMessageID),
		FileName:               nullableString(e.FileName),
		MimeType:               nullableString(e.MimeType),
		ObjectKey:              nullableString(e.ObjectKey),
		EditedAt:               e.EditedAt,
		DeletedForEveryoneAt:   e.DeletedForEveryoneAt,
		CreatedAt:              e.CreatedAt,
	}
}

func (r *messageRepoImpl) toEntity(m *models.MessageModel) *entity.MessageEntity {
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

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
