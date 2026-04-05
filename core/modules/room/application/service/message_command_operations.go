package service

import (
	"context"
	"errors"
	"strings"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
)

func (s *MessageCommandService) CreateMessage(ctx context.Context, accountID string, command apptypes.SendMessageCommand) (*apptypes.MessageResult, error) {
	return s.SendMessage(ctx, accountID, command)
}

func (s *MessageCommandService) SendMessage(ctx context.Context, accountID string, command apptypes.SendMessageCommand) (*apptypes.MessageResult, error) {
	roomID := strings.TrimSpace(command.RoomID)
	if roomID == "" {
		return nil, errors.New("room_id is required")
	}

	if _, room, err := requireRoomRole(ctx, s.repos.RoomRepository(), s.repos.RoomMemberRepository(), roomID, accountID); err != nil {
		return nil, err
	} else {
		messageType := normalizeMessageType(command.MessageType)
		if messageType == "" {
			messageType = "text"
		}

		content := strings.TrimSpace(command.Message)
		if messageType == "text" && content == "" {
			return nil, errors.New("message is required")
		}
		if (messageType == "image" || messageType == "file") && strings.TrimSpace(command.ObjectKey) == "" {
			return nil, errors.New("object_key is required for media messages")
		}

		now := time.Now().UTC()
		message := &entity.MessageEntity{
			ID:                     newUUID(),
			RoomID:                 roomID,
			SenderID:               accountID,
			Message:                content,
			MessageType:            messageType,
			ReplyToMessageID:       strings.TrimSpace(command.ReplyToMessageID),
			ForwardedFromMessageID: strings.TrimSpace(command.ForwardedFromMessageID),
			FileName:               strings.TrimSpace(command.FileName),
			FileSize:               command.FileSize,
			MimeType:               strings.TrimSpace(command.MimeType),
			ObjectKey:              strings.TrimSpace(command.ObjectKey),
			CreatedAt:              now,
		}

		if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
			if err := txRepos.MessageRepository().CreateMessage(ctx, message); err != nil {
				return err
			}
			if err := txRepos.MessageReadRepository().UpsertMessage(ctx, message); err != nil {
				return err
			}

			members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
			if err != nil {
				return err
			}
			for _, member := range members {
				if member.AccountID == accountID {
					continue
				}
				if err := txRepos.MessageReadRepository().UpsertMessageReceipt(ctx, message.ID, member.AccountID, "sent", nil, nil, now, now); err != nil {
					return err
				}
			}

			room.UpdatedAt = now
			if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
				return err
			}
			if err := txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now); err != nil {
				return err
			}

			payload, _ := currentAccountPayload(ctx)
			senderName := accountID
			senderEmail := ""
			if payload != nil {
				senderName = payload.Email
				senderEmail = payload.Email
			}
			return s.aggregateService.PublishMessageCreated(ctx, txRepos.RoomOutboxEventsRepository(), roomID, message.ID, accountID, senderName, senderEmail, message.Message, message.CreatedAt)
		}); err != nil {
			return nil, err
		}

		return buildMessageResult(ctx, s.repos, accountID, message)
	}
}

func (s *MessageCommandService) EditMessage(ctx context.Context, accountID, messageID string, command apptypes.EditMessageCommand) (*apptypes.MessageResult, error) {
	message, err := s.repos.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if message.SenderID != accountID {
		return nil, errors.New("cannot edit another user's message")
	}
	if message.MessageType == "system" {
		return nil, errors.New("system messages cannot be edited")
	}

	content := strings.TrimSpace(command.Message)
	if content == "" {
		return nil, errors.New("message is required")
	}

	now := time.Now().UTC()
	message.Message = content
	message.EditedAt = &now
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.MessageRepository().UpdateMessage(ctx, message); err != nil {
			return err
		}
		return txRepos.MessageReadRepository().UpsertMessage(ctx, message)
	}); err != nil {
		return nil, err
	}

	return buildMessageResult(ctx, s.repos, accountID, message)
}

func (s *MessageCommandService) DeleteMessage(ctx context.Context, accountID, messageID string, command apptypes.DeleteMessageCommand) error {
	message, err := s.repos.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	scope := strings.ToLower(strings.TrimSpace(command.Scope))
	if scope == "" {
		scope = "me"
	}
	now := time.Now().UTC()

	switch scope {
	case "everyone":
		if message.SenderID != accountID {
			return errors.New("cannot delete everyone for another user's message")
		}
		message.Message = ""
		message.DeletedForEveryoneAt = &now
		return s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
			if err := txRepos.MessageRepository().UpdateMessage(ctx, message); err != nil {
				return err
			}
			return txRepos.MessageReadRepository().UpsertMessage(ctx, message)
		})
	case "me":
		return s.repos.MessageReadRepository().UpsertMessageDeletion(ctx, messageID, accountID, now)
	default:
		return errors.New("scope must be one of: me, everyone")
	}
}

func (s *MessageCommandService) ForwardMessage(ctx context.Context, accountID, messageID string, command apptypes.ForwardMessageCommand) (*apptypes.MessageResult, error) {
	message, err := s.repos.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	return s.SendMessage(ctx, accountID, apptypes.SendMessageCommand{
		RoomID:                 strings.TrimSpace(command.TargetRoomID),
		Message:                message.Message,
		MessageType:            message.MessageType,
		ForwardedFromMessageID: message.ID,
		FileName:               message.FileName,
		FileSize:               message.FileSize,
		MimeType:               message.MimeType,
		ObjectKey:              message.ObjectKey,
	})
}

func (s *MessageCommandService) MarkMessageStatus(ctx context.Context, accountID, messageID string, command apptypes.MarkMessageStatusCommand) error {
	status := strings.ToLower(strings.TrimSpace(command.Status))
	if status != "delivered" && status != "seen" {
		return errors.New("status must be delivered or seen")
	}

	now := time.Now().UTC()
	message, err := s.repos.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if message.SenderID == accountID {
		return nil
	}

	deliveredAt := &now
	var seenAt *time.Time
	if status == "seen" {
		seenAt = &now
	}
	return s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.MessageReadRepository().UpsertMessageReceipt(ctx, messageID, accountID, status, deliveredAt, seenAt, now, now); err != nil {
			return err
		}

		member, err := txRepos.RoomMemberReadRepository().GetRoomMemberByAccount(ctx, message.RoomID, accountID)
		if err == nil && member != nil {
			member.LastDeliveredAt = &now
			if status == "seen" {
				member.LastReadAt = &now
			}
			member.UpdatedAt = now
			return txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, member)
		}

		return nil
	})
}
