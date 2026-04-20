package entity

import (
	"errors"
	"strings"
	"time"

	"wechat-clone/core/shared/pkg/stackErr"
)

const (
	MessageTypeText     = "text"
	MessageTypeSystem   = "system"
	MessageTypeImage    = "image"
	MessageTypeFile     = "file"
	MessageTypeSticker  = "sticker"
	MessageTypeTransfer = "transfer"
)

var (
	ErrMessageIDRequired           = errors.New("message_id is required")
	ErrMessageRoomRequired         = errors.New("room_id is required")
	ErrMessageSenderRequired       = errors.New("account_id is required")
	ErrMessageBodyRequired         = errors.New("message is required")
	ErrMessageTypeInvalid          = errors.New("message_type is invalid")
	ErrMessageObjectKeyRequired    = errors.New("object_key is required for media messages")
	ErrMessageCannotEditOther      = errors.New("cannot edit another user's message")
	ErrMessageCannotEditSystem     = errors.New("system messages cannot be edited")
	ErrMessageCannotDeleteEveryone = errors.New("cannot delete everyone for another user's message")
)

type MessageParams struct {
	Message                string
	MessageType            string
	Mentions               []MessageMention
	MentionAll             bool
	ReplyToMessageID       string
	ForwardedFromMessageID string
	FileName               string
	FileSize               int64
	MimeType               string
	ObjectKey              string
}

func NewMessage(id, roomID, senderID string, params MessageParams, now time.Time) (*MessageEntity, error) {
	id = strings.TrimSpace(id)
	roomID = strings.TrimSpace(roomID)
	senderID = strings.TrimSpace(senderID)
	messageType := NormalizeMessageType(params.MessageType)
	content := strings.TrimSpace(params.Message)
	objectKey := strings.TrimSpace(params.ObjectKey)
	mentions, err := NormalizeMessageMentions(params.Mentions)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	switch {
	case id == "":
		return nil, stackErr.Error(ErrMessageIDRequired)
	case roomID == "":
		return nil, stackErr.Error(ErrMessageRoomRequired)
	case senderID == "":
		return nil, stackErr.Error(ErrMessageSenderRequired)
	case messageType == "":
		return nil, stackErr.Error(ErrMessageTypeInvalid)
	case messageType == MessageTypeText && content == "":
		return nil, stackErr.Error(ErrMessageBodyRequired)
	case (messageType == MessageTypeImage || messageType == MessageTypeFile || messageType == MessageTypeSticker) && objectKey == "":
		return nil, stackErr.Error(ErrMessageObjectKeyRequired)
	}

	return &MessageEntity{
		ID:                     id,
		RoomID:                 roomID,
		SenderID:               senderID,
		Message:                content,
		MessageType:            messageType,
		Mentions:               mentions,
		Reactions:              nil,
		MentionAll:             params.MentionAll,
		ReplyToMessageID:       strings.TrimSpace(params.ReplyToMessageID),
		ForwardedFromMessageID: strings.TrimSpace(params.ForwardedFromMessageID),
		FileName:               strings.TrimSpace(params.FileName),
		FileSize:               params.FileSize,
		MimeType:               strings.TrimSpace(params.MimeType),
		ObjectKey:              objectKey,
		CreatedAt:              normalizeRoomTime(now),
	}, nil
}

func NormalizeMessageType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", MessageTypeText:
		return MessageTypeText
	case MessageTypeSystem:
		return MessageTypeSystem
	case MessageTypeImage:
		return MessageTypeImage
	case MessageTypeFile:
		return MessageTypeFile
	case MessageTypeSticker:
		return MessageTypeSticker
	case MessageTypeTransfer:
		return MessageTypeTransfer
	default:
		return ""
	}
}

func (m *MessageEntity) Edit(actorID, content string, editedAt time.Time) error {
	if strings.TrimSpace(actorID) != strings.TrimSpace(m.SenderID) {
		return stackErr.Error(ErrMessageCannotEditOther)
	}
	if NormalizeMessageType(m.MessageType) == MessageTypeSystem {
		return stackErr.Error(ErrMessageCannotEditSystem)
	}
	if content = strings.TrimSpace(content); content == "" {
		return stackErr.Error(ErrMessageBodyRequired)
	}

	now := normalizeRoomTime(editedAt)
	m.Message = content
	m.EditedAt = &now
	return nil
}

func (m *MessageEntity) DeleteForEveryone(actorID string, deletedAt time.Time) error {
	if strings.TrimSpace(actorID) != strings.TrimSpace(m.SenderID) {
		return ErrMessageCannotDeleteEveryone
	}

	now := normalizeRoomTime(deletedAt)
	m.Message = ""
	m.DeletedForEveryoneAt = &now
	return nil
}
