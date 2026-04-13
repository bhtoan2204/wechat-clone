package support

import (
	"context"
	"errors"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/samber/lo"
)

func BuildConversationResultFromState(
	ctx context.Context,
	baseRepo roomrepos.Repos,
	viewerID string,
	room *entity.Room,
	members []*entity.RoomMemberEntity,
	lastMessage *entity.MessageEntity,
	includeMembers bool,
) (*apptypes.ConversationResult, error) {
	if room == nil {
		return nil, stackErr.Error(errors.New("room is required"))
	}

	var viewerMember *entity.RoomMemberEntity
	accountIDs := make([]string, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}
		accountIDs = append(accountIDs, member.AccountID)
		if member.AccountID == viewerID {
			viewerMember = member
		}
	}
	if viewerMember == nil {
		return nil, stackErr.Error(errors.New("viewer is not a member of this room"))
	}

	accountProjections, err := baseRepo.RoomAccountProjectionRepository().ListByAccountIDs(ctx, lo.Uniq(accountIDs))
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountMap := lo.SliceToMap(accountProjections, func(acc *entity.AccountEntity) (string, *entity.AccountEntity) {
		return acc.AccountID, acc
	})

	name := room.Name
	if string(room.RoomType) == "direct" {
		if otherMember, found := lo.Find(members, func(member *entity.RoomMemberEntity) bool {
			return member != nil && member.AccountID != viewerID
		}); found {
			if acc, ok := accountMap[otherMember.AccountID]; ok && acc != nil {
				name = firstNonEmpty(acc.DisplayName, acc.Username, otherMember.DisplayName, otherMember.Username, otherMember.AccountID)
			} else {
				name = firstNonEmpty(otherMember.DisplayName, otherMember.Username, otherMember.AccountID)
			}
		}
	}

	result := &apptypes.ConversationResult{
		RoomID:          room.ID,
		Name:            name,
		Description:     room.Description,
		RoomType:        string(room.RoomType),
		OwnerID:         room.OwnerID,
		PinnedMessageID: room.PinnedMessageID,
		MemberCount:     len(members),
		UnreadCount:     0,
		CreatedAt:       room.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       room.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if includeMembers {
		result.Members = make([]apptypes.ConversationMemberResult, 0, len(members))
		for _, member := range members {
			if member == nil {
				continue
			}

			item := apptypes.ConversationMemberResult{
				AccountID:       member.AccountID,
				Role:            string(member.Role),
				DisplayName:     member.DisplayName,
				Username:        member.Username,
				AvatarObjectKey: member.AvatarObjectKey,
			}

			if acc, ok := accountMap[member.AccountID]; ok && acc != nil {
				item.DisplayName = firstNonEmpty(acc.DisplayName, item.DisplayName)
				item.Username = firstNonEmpty(acc.Username, item.Username)
				item.AvatarObjectKey = firstNonEmpty(acc.AvatarObjectKey, item.AvatarObjectKey)
			}

			result.Members = append(result.Members, item)
		}
	}

	if lastMessage != nil {
		messageResult, err := BuildMessageResultFromState(ctx, baseRepo, viewerID, lastMessage)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		result.LastMessage = messageResult
	}

	return result, nil
}

func BuildMessageResultFromState(
	ctx context.Context,
	baseRepo roomrepos.Repos,
	viewerID string,
	message *entity.MessageEntity,
) (*apptypes.MessageResult, error) {
	if message == nil {
		return nil, stackErr.Error(errors.New("message is required"))
	}

	status := "sent"
	if message.SenderID != viewerID {
		status = "delivered"
	}

	result := &apptypes.MessageResult{
		ID:                     message.ID,
		RoomID:                 message.RoomID,
		SenderID:               message.SenderID,
		Message:                message.Message,
		MessageType:            message.MessageType,
		Status:                 status,
		MentionAll:             message.MentionAll,
		ReplyToMessageID:       message.ReplyToMessageID,
		ForwardedFromMessageID: message.ForwardedFromMessageID,
		FileName:               message.FileName,
		FileSize:               message.FileSize,
		MimeType:               message.MimeType,
		ObjectKey:              message.ObjectKey,
		DeletedForEveryone:     message.DeletedForEveryoneAt != nil,
		CreatedAt:              message.CreatedAt.UTC().Format(time.RFC3339),
	}

	if message.EditedAt != nil {
		result.EditedAt = message.EditedAt.UTC().Format(time.RFC3339)
	}
	if message.DeletedForEveryoneAt != nil {
		result.Message = ""
	}

	if len(message.Mentions) > 0 {
		result.Mentions = make([]apptypes.MessageMentionResult, 0, len(message.Mentions))
		for _, mention := range message.Mentions {
			result.Mentions = append(result.Mentions, apptypes.MessageMentionResult{
				AccountID:   mention.AccountID,
				DisplayName: mention.DisplayName,
				Username:    mention.Username,
			})
		}
	}

	if message.ReplyToMessageID != "" {
		replyTo, err := baseRepo.MessageRepository().GetMessageByID(ctx, message.ReplyToMessageID)
		if err == nil && replyTo != nil {
			result.ReplyTo = &apptypes.MessagePreviewResult{
				ID:          replyTo.ID,
				SenderID:    replyTo.SenderID,
				Message:     replyTo.Message,
				MessageType: replyTo.MessageType,
			}
		}
	}

	if message.ForwardedFromMessageID != "" {
		forwardedFrom, err := baseRepo.MessageRepository().GetMessageByID(ctx, message.ForwardedFromMessageID)
		if err == nil && forwardedFrom != nil {
			result.ForwardedFrom = &apptypes.MessagePreviewResult{
				ID:          forwardedFrom.ID,
				SenderID:    forwardedFrom.SenderID,
				Message:     forwardedFrom.Message,
				MessageType: forwardedFrom.MessageType,
			}
		}
	}

	return result, nil
}
