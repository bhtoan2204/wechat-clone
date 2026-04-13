package support

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-socket/core/modules/room/application/projection"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/infra/projection/cassandra/views"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

func BuildRoomResult(room *views.RoomView) *apptypes.RoomResult {
	if room == nil {
		return nil
	}

	return &apptypes.RoomResult{
		ID:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		RoomType:    strings.TrimSpace(room.RoomType),
		OwnerID:     room.OwnerID,
		CreatedAt:   room.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   room.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func BuildConversationResult(
	ctx context.Context,
	readRepos projection.QueryRepos,
	viewerID string,
	room *views.RoomView,
	includeMembers bool,
) (*apptypes.ConversationResult, error) {
	if room == nil {
		return nil, stackErr.Error(errors.New("room is required"))
	}

	members, err := readRepos.RoomMemberReadRepository().ListRoomMembers(ctx, room.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var viewerMember *views.RoomMemberView
	for _, member := range members {
		if member != nil && strings.TrimSpace(member.AccountID) == strings.TrimSpace(viewerID) {
			viewerMember = member
			break
		}
	}
	if viewerMember == nil {
		return nil, stackErr.Error(errors.New("viewer is not a member of this room"))
	}

	var (
		unreadCount int64
		lastMessage *views.MessageView
	)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var runErr error
		unreadCount, runErr = readRepos.MessageReadRepository().CountUnreadMessages(
			egCtx,
			room.ID,
			viewerID,
			viewerMember.LastReadAt,
		)
		if runErr != nil {
			return stackErr.Error(runErr)
		}
		return nil
	})
	eg.Go(func() error {
		var runErr error
		lastMessage, runErr = readRepos.MessageReadRepository().GetLastMessage(egCtx, room.ID)
		if runErr != nil {
			return stackErr.Error(runErr)
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, stackErr.Error(err)
	}

	name := room.Name
	if strings.EqualFold(strings.TrimSpace(room.RoomType), "direct") {
		if otherMember, found := lo.Find(members, func(member *views.RoomMemberView) bool {
			return member != nil && strings.TrimSpace(member.AccountID) != strings.TrimSpace(viewerID)
		}); found {
			name = firstNonEmpty(otherMember.DisplayName, otherMember.Username, otherMember.AccountID)
		}
	}

	result := &apptypes.ConversationResult{
		RoomID:          room.ID,
		Name:            name,
		Description:     room.Description,
		RoomType:        strings.TrimSpace(room.RoomType),
		OwnerID:         room.OwnerID,
		PinnedMessageID: derefString(room.PinnedMessageID),
		MemberCount:     len(members),
		UnreadCount:     unreadCount,
		CreatedAt:       room.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       room.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if includeMembers {
		result.Members = make([]apptypes.ConversationMemberResult, 0, len(members))
		for _, member := range members {
			if member == nil {
				continue
			}
			result.Members = append(result.Members, apptypes.ConversationMemberResult{
				AccountID:       member.AccountID,
				Role:            strings.TrimSpace(member.Role),
				DisplayName:     strings.TrimSpace(member.DisplayName),
				Username:        strings.TrimSpace(member.Username),
				AvatarObjectKey: strings.TrimSpace(member.AvatarObjectKey),
			})
		}
	}

	if lastMessage != nil {
		result.LastMessage, err = BuildMessageResult(ctx, readRepos, viewerID, lastMessage)
		if err != nil {
			return nil, stackErr.Error(err)
		}
	}

	return result, nil
}

func BuildMessageResult(
	ctx context.Context,
	readRepos projection.QueryRepos,
	viewerID string,
	message *views.MessageView,
) (*apptypes.MessageResult, error) {
	if message == nil {
		return nil, stackErr.Error(errors.New("message is required"))
	}

	status := "sent"
	if strings.TrimSpace(message.SenderID) == strings.TrimSpace(viewerID) {
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
		if err == nil && strings.TrimSpace(receiptStatus) != "" {
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

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
