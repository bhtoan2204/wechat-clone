package service

import (
	"context"
	"errors"
	"time"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
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

func buildConversationResult(ctx context.Context, readRepos repos.QueryRepos, viewerID string, room *entity.Room, includeMembers bool) (*apptypes.ConversationResult, error) {
	members, err := readRepos.RoomMemberReadRepository().ListRoomMembers(ctx, room.ID)
	if err != nil {
		return nil, err
	}

	var viewerMember *entity.RoomMemberEntity
	for _, member := range members {
		if member.AccountID == viewerID {
			viewerMember = member
			break
		}
	}
	if viewerMember == nil {
		return nil, errors.New("viewer is not a member of this room")
	}

	unreadCount, err := readRepos.MessageReadRepository().CountUnreadMessages(ctx, room.ID, viewerID, viewerMember.LastReadAt)
	if err != nil {
		return nil, err
	}

	name := room.Name
	if string(room.RoomType) == "direct" {
		for _, member := range members {
			if member.AccountID != viewerID {
				name = member.AccountID
				break
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
		result.Members = make([]apptypes.ConversationMemberResult, 0, len(members))
		for _, member := range members {
			result.Members = append(result.Members, apptypes.ConversationMemberResult{
				AccountID: member.AccountID,
				Role:      string(member.Role),
			})
		}
	}

	lastMessage, err := readRepos.MessageReadRepository().GetLastMessage(ctx, room.ID)
	if err == nil && lastMessage != nil {
		result.LastMessage, err = buildMessageResult(ctx, readRepos, viewerID, lastMessage)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func buildMessageResult(ctx context.Context, readRepos repos.QueryRepos, viewerID string, message *entity.MessageEntity) (*apptypes.MessageResult, error) {
	status := "sent"
	if message.SenderID == viewerID {
		seenCount, err := readRepos.MessageReadRepository().CountMessageReceiptsByStatus(ctx, message.ID, "seen")
		if err != nil {
			return nil, err
		}
		if seenCount > 0 {
			status = "seen"
		} else {
			deliveredCount, err := readRepos.MessageReadRepository().CountMessageReceiptsByStatus(ctx, message.ID, "delivered")
			if err != nil {
				return nil, err
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
