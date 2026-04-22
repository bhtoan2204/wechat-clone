package repository

import (
	"context"
	"errors"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/modules/room/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"gorm.io/gorm"
)

type messageRepoImpl struct {
	db *gorm.DB
}

func NewMessageRepoImpl(db *gorm.DB) repos.MessageRepository {
	return &messageRepoImpl{db: db}
}

func (r *messageRepoImpl) CreateMessage(ctx context.Context, message *entity.MessageEntity) error {
	m, err := r.toModel(message)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return stackErr.Error(err)
	}
	return nil
}

func (r *messageRepoImpl) UpdateMessage(ctx context.Context, message *entity.MessageEntity) error {
	m, err := r.toModel(message)
	if err != nil {
		return stackErr.Error(err)
	}
	return r.db.WithContext(ctx).Model(&models.MessageModel{}).Where("id = ?", message.ID).Updates(map[string]interface{}{
		"room_id":                   m.RoomID,
		"sender_id":                 m.SenderID,
		"message":                   m.Message,
		"message_type":              m.MessageType,
		"mentions_json":             m.MentionsJSON,
		"reactions_json":            m.ReactionsJSON,
		"mention_all":               m.MentionAll,
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}
	entityMessage, err := r.toEntity(&m)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return entityMessage, nil
}

func (r *messageRepoImpl) GetLastMessageByRoomID(ctx context.Context, roomID string) (*entity.MessageEntity, error) {
	var m models.MessageModel
	if err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at DESC, id DESC").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	entityMessage, err := r.toEntity(&m)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return entityMessage, nil
}

func (r *messageRepoImpl) toModel(e *entity.MessageEntity) (*models.MessageModel, error) {
	mentionsJSON, err := marshalMessageMentions(e.Mentions)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	reactionsJSON, err := marshalMessageReactions(e.Reactions)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &models.MessageModel{
		ID:                     e.ID,
		RoomID:                 e.RoomID,
		SenderID:               e.SenderID,
		Message:                e.Message,
		MessageType:            e.MessageType,
		MentionsJSON:           mentionsJSON,
		ReactionsJSON:          reactionsJSON,
		MentionAll:             utils.BoolToSmallInt(e.MentionAll),
		ReplyToMessageID:       utils.NullableString(e.ReplyToMessageID),
		ForwardedFromMessageID: utils.NullableString(e.ForwardedFromMessageID),
		FileName:               utils.NullableString(e.FileName),
		FileSize:               utils.Int64Ptr(e.FileSize),
		MimeType:               utils.NullableString(e.MimeType),
		ObjectKey:              utils.NullableString(e.ObjectKey),
		EditedAt:               e.EditedAt,
		DeletedForEveryoneAt:   e.DeletedForEveryoneAt,
		CreatedAt:              e.CreatedAt,
	}, nil
}

func (r *messageRepoImpl) toEntity(m *models.MessageModel) (*entity.MessageEntity, error) {
	mentions, err := unmarshalMessageMentions(m.MentionsJSON)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	reactions, err := unmarshalMessageReactions(m.ReactionsJSON)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var fileSize int64
	if m.FileSize != nil {
		fileSize = *m.FileSize
	}

	return &entity.MessageEntity{
		ID:                     m.ID,
		RoomID:                 m.RoomID,
		SenderID:               m.SenderID,
		Message:                m.Message,
		MessageType:            m.MessageType,
		Mentions:               mentions,
		Reactions:              reactions,
		MentionAll:             m.MentionAll == 1,
		ReplyToMessageID:       utils.StringValue(m.ReplyToMessageID),
		ForwardedFromMessageID: utils.StringValue(m.ForwardedFromMessageID),
		FileName:               utils.StringValue(m.FileName),
		FileSize:               fileSize,
		MimeType:               utils.StringValue(m.MimeType),
		ObjectKey:              utils.StringValue(m.ObjectKey),
		EditedAt:               m.EditedAt,
		DeletedForEveryoneAt:   m.DeletedForEveryoneAt,
		CreatedAt:              m.CreatedAt,
	}, nil
}
