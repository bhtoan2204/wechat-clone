package support

import (
	"errors"
	"time"

	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/samber/lo"
)

func BuildConversationResultFromState(
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
	for _, member := range members {
		if member == nil {
			continue
		}
		if member.AccountID == viewerID {
			viewerMember = member
			break
		}
	}
	if viewerMember == nil {
		return nil, stackErr.Error(ErrViewerNotMemberOfRoom)
	}

	name := room.Name
	if string(room.RoomType) == "direct" {
		if otherMember, found := lo.Find(members, func(member *entity.RoomMemberEntity) bool {
			return member != nil && member.AccountID != viewerID
		}); found {
			name = firstNonEmpty(otherMember.DisplayName, otherMember.Username, otherMember.AccountID)
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
		result.Members = lo.FilterMap(members, func(member *entity.RoomMemberEntity, _ int) (apptypes.ConversationMemberResult, bool) {
			if member == nil {
				return apptypes.ConversationMemberResult{}, false
			}

			item := apptypes.ConversationMemberResult{
				AccountID:       member.AccountID,
				Role:            string(member.Role),
				DisplayName:     member.DisplayName,
				Username:        member.Username,
				AvatarObjectKey: member.AvatarObjectKey,
			}

			return item, true
		})
	}

	if lastMessage != nil {
		messageResult, err := BuildMessageResultFromState(viewerID, lastMessage)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		result.LastMessage = messageResult
	}

	return result, nil
}

func BuildMessageResultFromState(
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
		Reactions:              buildStateMessageReactionResults(viewerID, message.Reactions),
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
		result.Mentions = lo.Map(message.Mentions, func(mention entity.MessageMention, _ int) apptypes.MessageMentionResult {
			return apptypes.MessageMentionResult{
				AccountID:   mention.AccountID,
				DisplayName: mention.DisplayName,
				Username:    mention.Username,
			}
		})
	}

	return result, nil
}

func buildStateMessageReactionResults(viewerID string, items []entity.MessageReaction) []apptypes.MessageReactionResult {
	if len(items) == 0 {
		return nil
	}

	type groupedReaction struct {
		emoji      string
		accountIDs []string
		byMe       bool
	}

	groups := make(map[string]*groupedReaction, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		emoji := item.Emoji
		accountID := item.AccountID
		if emoji == "" || accountID == "" {
			continue
		}

		group, exists := groups[emoji]
		if !exists {
			group = &groupedReaction{emoji: emoji}
			groups[emoji] = group
			order = append(order, emoji)
		}

		group.accountIDs = append(group.accountIDs, accountID)
		if accountID == viewerID {
			group.byMe = true
		}
	}

	results := make([]apptypes.MessageReactionResult, 0, len(order))
	for _, emoji := range order {
		group := groups[emoji]
		if group == nil || len(group.accountIDs) == 0 {
			continue
		}
		results = append(results, apptypes.MessageReactionResult{
			Emoji:       group.emoji,
			Count:       len(group.accountIDs),
			ReactedByMe: group.byMe,
			AccountIDs:  group.accountIDs,
		})
	}
	if len(results) == 0 {
		return nil
	}
	return results
}
