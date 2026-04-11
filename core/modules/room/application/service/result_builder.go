package service

import (
	"context"
	"errors"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

func buildRoomResult(room *entity.Room) *apptypes.RoomResult {
	if room == nil {
		return nil
	}

	return &apptypes.RoomResult{
		ID:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		RoomType:    string(room.RoomType),
		OwnerID:     room.OwnerID,
		CreatedAt:   room.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   room.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func buildConversationResult(
	ctx context.Context,
	readRepos repos.QueryRepos,
	viewerID string,
	room *entity.Room,
	includeMembers bool,
) (*apptypes.ConversationResult, error) {
	members, err := readRepos.RoomMemberReadRepository().ListRoomMembers(ctx, room.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var viewerMember *entity.RoomMemberEntity
	for _, member := range members {
		if member.AccountID == viewerID {
			viewerMember = member
		}
	}
	if viewerMember == nil {
		return nil, stackErr.Error(errors.New("viewer is not a member of this room"))
	}

	accountIDs := lo.Map(members, func(member *entity.RoomMemberEntity, _ int) string {
		return member.AccountID
	})

	var (
		accountProjections []*entity.AccountEntity
		unreadCount        int64
		lastMessage        *entity.MessageEntity
	)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		var err error
		accountProjections, err = readRepos.RoomAccountProjectionRepository().ListByAccountIDs(egCtx, accountIDs)
		if err != nil {
			return stackErr.Error(err)
		}
		return nil
	})

	eg.Go(func() error {
		var err error
		unreadCount, err = readRepos.MessageReadRepository().CountUnreadMessages(
			egCtx,
			room.ID,
			viewerID,
			viewerMember.LastReadAt,
		)
		if err != nil {
			return stackErr.Error(err)
		}
		return nil
	})

	eg.Go(func() error {
		var err error
		lastMessage, err = readRepos.MessageReadRepository().GetLastMessage(egCtx, room.ID)
		if err != nil {
			return stackErr.Error(err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, stackErr.Error(err)
	}

	accountMap := lo.SliceToMap(accountProjections, func(acc *entity.AccountEntity) (string, *entity.AccountEntity) {
		return acc.AccountID, acc
	})

	name := room.Name
	if string(room.RoomType) == "direct" {
		if otherMember, found := lo.Find(members, func(member *entity.RoomMemberEntity) bool {
			return member.AccountID != viewerID
		}); found {
			if acc, ok := accountMap[otherMember.AccountID]; ok {
				switch {
				case acc.DisplayName != "":
					name = acc.DisplayName
				case acc.Username != "":
					name = acc.Username
				default:
					name = otherMember.AccountID
				}
			} else {
				name = otherMember.AccountID
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
		UnreadCount:     unreadCount,
		CreatedAt:       room.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       room.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if includeMembers {
		result.Members = lo.Map(members, func(member *entity.RoomMemberEntity, _ int) apptypes.ConversationMemberResult {
			item := apptypes.ConversationMemberResult{
				AccountID: member.AccountID,
				Role:      string(member.Role),
			}

			if acc, ok := accountMap[member.AccountID]; ok {
				item.DisplayName = acc.DisplayName
				item.Username = acc.Username
				item.AvatarObjectKey = acc.AvatarObjectKey
			}

			return item
		})
	}

	if lastMessage != nil {
		result.LastMessage, err = buildMessageResult(ctx, readRepos, viewerID, lastMessage)
		if err != nil {
			return nil, stackErr.Error(err)
		}
	}

	return result, nil
}

func buildMessageResult(ctx context.Context, readRepos repos.QueryRepos, viewerID string, message *entity.MessageEntity) (*apptypes.MessageResult, error) {
	status := "sent"
	if message.SenderID == viewerID {
		seenCount, err := readRepos.MessageReadRepository().CountMessageReceiptsByStatus(ctx, message.ID, "seen")
		if err != nil {
			return nil, stackErr.Error(err)
		}
		if seenCount > 0 {
			status = "seen"
		} else {
			deliveredCount, err := readRepos.MessageReadRepository().CountMessageReceiptsByStatus(ctx, message.ID, "delivered")
			if err != nil {
				return nil, stackErr.Error(err)
			}
			if deliveredCount > 0 {
				status = "delivered"
			}
		}
	} else {
		receiptStatus, _, _, err := readRepos.MessageReadRepository().GetMessageReceipt(ctx, message.ID, viewerID)
		if err == nil && receiptStatus != "" {
			status = receiptStatus
		}
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
		replyTo, err := readRepos.MessageReadRepository().GetMessageByID(ctx, message.ReplyToMessageID)
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
		forwardedFrom, err := readRepos.MessageReadRepository().GetMessageByID(ctx, message.ForwardedFromMessageID)
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
