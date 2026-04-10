package service

import (
	"context"
	"errors"
	"strings"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/stackErr"
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
		return nil, stackErr.Error(err)
	} else {
		now := time.Now().UTC()
		message, err := entity.NewMessage(newUUID(), roomID, accountID, entity.MessageParams{
			Message:                command.Message,
			MessageType:            command.MessageType,
			ReplyToMessageID:       command.ReplyToMessageID,
			ForwardedFromMessageID: command.ForwardedFromMessageID,
			FileName:               command.FileName,
			FileSize:               command.FileSize,
			MimeType:               command.MimeType,
			ObjectKey:              command.ObjectKey,
		}, now)
		if err != nil {
			return nil, stackErr.Error(err)
		}

		if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
			if err := txRepos.MessageRepository().CreateMessage(ctx, message); err != nil {
				return stackErr.Error(err)
			}
			if err := txRepos.MessageReadRepository().UpsertMessage(ctx, message); err != nil {
				return stackErr.Error(err)
			}

			members, err := txRepos.RoomMemberReadRepository().ListRoomMembers(ctx, roomID)
			if err != nil {
				return stackErr.Error(err)
			}
			for _, member := range members {
				if member.AccountID == accountID {
					continue
				}
				if err := txRepos.MessageReadRepository().UpsertMessageReceipt(ctx, message.ID, member.AccountID, "sent", nil, nil, now, now); err != nil {
					return stackErr.Error(err)
				}
			}

			room.Touch(now)
			if err := txRepos.RoomRepository().UpdateRoom(ctx, room); err != nil {
				return stackErr.Error(err)
			}
			if err := txRepos.RoomReadRepository().UpdateRoomStats(ctx, roomID, len(members), message, now); err != nil {
				return stackErr.Error(err)
			}

			actor, _ := currentActor(ctx)
			senderName := accountID
			senderEmail := ""
			if actor != nil && actor.Email != "" {
				senderName = actor.Email
				senderEmail = actor.Email
			}
			return s.aggregateService.PublishMessageCreated(ctx, txRepos.RoomOutboxEventsRepository(), roomID, message.ID, accountID, senderName, senderEmail, message.Message, message.CreatedAt)
		}); err != nil {
			return nil, stackErr.Error(err)
		}

		return buildMessageResult(ctx, s.repos, accountID, message)
	}
}

func (s *MessageCommandService) EditMessage(ctx context.Context, accountID, messageID string, command apptypes.EditMessageCommand) (*apptypes.MessageResult, error) {
	message, err := s.repos.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := message.Edit(accountID, command.Message, time.Now().UTC()); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.MessageRepository().UpdateMessage(ctx, message); err != nil {
			return err
		}
		return txRepos.MessageReadRepository().UpsertMessage(ctx, message)
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return buildMessageResult(ctx, s.repos, accountID, message)
}

func (s *MessageCommandService) DeleteMessage(ctx context.Context, accountID, messageID string, command apptypes.DeleteMessageCommand) error {
	message, err := s.repos.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return stackErr.Error(err)
	}

	scope := strings.ToLower(strings.TrimSpace(command.Scope))
	if scope == "" {
		scope = "me"
	}
	now := time.Now().UTC()

	switch scope {
	case "everyone":
		if err := message.DeleteForEveryone(accountID, now); err != nil {
			return err
		}
		return s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
			if err := txRepos.MessageRepository().UpdateMessage(ctx, message); err != nil {
				return stackErr.Error(err)
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
		return nil, stackErr.Error(err)
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
	message, err := s.repos.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return stackErr.Error(err)
	}
	if !message.CanBeMarkedBy(accountID) {
		return nil
	}

	status, err := entity.NormalizeReceiptStatus(command.Status)
	if err != nil {
		return stackErr.Error(err)
	}

	now := time.Now().UTC()
	deliveredAt := &now
	var seenAt *time.Time
	return s.repos.WithTransaction(ctx, func(txRepos repos.Repos) error {
		member, err := txRepos.RoomMemberReadRepository().GetRoomMemberByAccount(ctx, message.RoomID, accountID)
		if err == nil && member != nil {
			var applyErr error
			status, deliveredAt, seenAt, applyErr = member.ApplyReceiptStatus(status, now)
			if applyErr != nil {
				return applyErr
			}
		}

		if err := txRepos.MessageReadRepository().UpsertMessageReceipt(ctx, messageID, accountID, status, deliveredAt, seenAt, now, now); err != nil {
			return err
		}
		if err == nil && member != nil {
			return txRepos.RoomMemberReadRepository().UpsertRoomMember(ctx, member)
		}
		return nil
	})
}
