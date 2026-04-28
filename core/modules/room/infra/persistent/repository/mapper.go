package repository

import (
	"encoding/json"
	"strings"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/infra/persistent/models"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"
)

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

func marshalMessageReactions(items []entity.MessageReaction) (string, error) {
	if len(items) == 0 {
		return "[]", nil
	}

	data, err := json.Marshal(items)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return string(data), nil
}

func unmarshalMessageReactions(raw string) ([]entity.MessageReaction, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var items []entity.MessageReaction
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, stackErr.Error(err)
	}
	return entity.NormalizeMessageReactions(items)
}

func marshalMessageMentions(mentions []entity.MessageMention) (string, error) {
	if len(mentions) == 0 {
		return "[]", nil
	}

	data, err := json.Marshal(mentions)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return string(data), nil
}

func unmarshalMessageMentions(raw string) ([]entity.MessageMention, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var mentions []entity.MessageMention
	if err := json.Unmarshal([]byte(raw), &mentions); err != nil {
		return nil, stackErr.Error(err)
	}
	return entity.NormalizeMessageMentions(mentions)
}
