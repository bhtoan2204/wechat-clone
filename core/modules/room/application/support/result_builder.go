package support

import (
	"context"
	"errors"
	"strings"
	"time"

	"wechat-clone/core/modules/room/application/projection"
	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/modules/room/infra/projection/cassandra/views"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

type ConversationBuildInput struct {
	ViewerID       string
	Room           *views.RoomView
	IncludeMembers bool
}

type MessageBuildInput struct {
	ViewerID string
	Message  *views.MessageView
}

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
	return newConversationQueryBuilder(readRepos).Build(ctx, ConversationBuildInput{
		ViewerID:       viewerID,
		Room:           room,
		IncludeMembers: includeMembers,
	})
}

func BuildMessageResult(
	ctx context.Context,
	readRepos projection.QueryRepos,
	viewerID string,
	message *views.MessageView,
) (*apptypes.MessageResult, error) {
	return newMessageQueryBuilder(readRepos).Build(ctx, MessageBuildInput{
		ViewerID: viewerID,
		Message:  message,
	})
}

type conversationQueryBuilder struct {
	readRepos      projection.QueryRepos
	messageBuilder *messageQueryBuilder
}

func newConversationQueryBuilder(readRepos projection.QueryRepos) *conversationQueryBuilder {
	return &conversationQueryBuilder{
		readRepos:      readRepos,
		messageBuilder: newMessageQueryBuilder(readRepos),
	}
}

func (b *conversationQueryBuilder) Build(ctx context.Context, input ConversationBuildInput) (*apptypes.ConversationResult, error) {
	if input.Room == nil {
		return nil, stackErr.Error(errors.New("room is required"))
	}

	members, err := b.readRepos.RoomMemberReadRepository().ListRoomMembers(ctx, input.Room.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	viewerMember, found := lo.Find(members, func(member *views.RoomMemberView) bool {
		return member != nil && strings.TrimSpace(member.AccountID) == strings.TrimSpace(input.ViewerID)
	})
	if !found || viewerMember == nil {
		return nil, stackErr.Error(ErrViewerNotMemberOfRoom)
	}

	unreadCount, lastMessage, err := b.loadConversationState(ctx, input.Room.ID, input.ViewerID, viewerMember.LastReadAt)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	result := &apptypes.ConversationResult{
		RoomID:          input.Room.ID,
		Name:            b.resolveConversationName(input.Room, members, input.ViewerID),
		Description:     input.Room.Description,
		RoomType:        strings.TrimSpace(input.Room.RoomType),
		OwnerID:         input.Room.OwnerID,
		PinnedMessageID: utils.DerefString(input.Room.PinnedMessageID),
		MemberCount:     len(members),
		UnreadCount:     unreadCount,
		CreatedAt:       input.Room.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       input.Room.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if input.IncludeMembers {
		result.Members = b.mapConversationMembers(members)
	}

	if lastMessage != nil {
		result.LastMessage, err = b.messageBuilder.Build(ctx, MessageBuildInput{
			ViewerID: input.ViewerID,
			Message:  lastMessage,
		})
		if err != nil {
			return nil, stackErr.Error(err)
		}
	}

	return result, nil
}

func (b *conversationQueryBuilder) loadConversationState(
	ctx context.Context,
	roomID string,
	viewerID string,
	lastReadAt *time.Time,
) (int64, *views.MessageView, error) {
	var (
		unreadCount int64
		lastMessage *views.MessageView
	)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var runErr error
		unreadCount, runErr = b.readRepos.MessageReadRepository().CountUnreadMessages(
			egCtx,
			roomID,
			viewerID,
			lastReadAt,
		)
		if runErr != nil {
			return stackErr.Error(runErr)
		}
		return nil
	})
	eg.Go(func() error {
		var runErr error
		lastMessage, runErr = b.readRepos.MessageReadRepository().GetLastMessage(egCtx, roomID)
		if runErr != nil {
			return stackErr.Error(runErr)
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return 0, nil, stackErr.Error(err)
	}

	return unreadCount, lastMessage, nil
}

func (b *conversationQueryBuilder) resolveConversationName(room *views.RoomView, members []*views.RoomMemberView, viewerID string) string {
	name := room.Name
	if !strings.EqualFold(strings.TrimSpace(room.RoomType), "direct") {
		return name
	}

	otherMember, found := lo.Find(members, func(member *views.RoomMemberView) bool {
		return member != nil && strings.TrimSpace(member.AccountID) != strings.TrimSpace(viewerID)
	})
	if !found || otherMember == nil {
		return name
	}

	return firstNonEmpty(otherMember.DisplayName, otherMember.Username, otherMember.AccountID)
}

func (b *conversationQueryBuilder) mapConversationMembers(members []*views.RoomMemberView) []apptypes.ConversationMemberResult {
	results := lo.FilterMap(members, func(member *views.RoomMemberView, _ int) (apptypes.ConversationMemberResult, bool) {
		if member == nil {
			return apptypes.ConversationMemberResult{}, false
		}

		return apptypes.ConversationMemberResult{
			AccountID:       member.AccountID,
			Role:            strings.TrimSpace(member.Role),
			DisplayName:     strings.TrimSpace(member.DisplayName),
			Username:        strings.TrimSpace(member.Username),
			AvatarObjectKey: strings.TrimSpace(member.AvatarObjectKey),
		}, true
	})
	if len(results) == 0 {
		return nil
	}
	return results
}

type messageQueryBuilder struct {
	readRepos projection.QueryRepos
}

func newMessageQueryBuilder(readRepos projection.QueryRepos) *messageQueryBuilder {
	return &messageQueryBuilder{readRepos: readRepos}
}

func (b *messageQueryBuilder) Build(ctx context.Context, input MessageBuildInput) (*apptypes.MessageResult, error) {
	if input.Message == nil {
		return nil, stackErr.Error(errors.New("message is required"))
	}

	status, err := b.resolveViewerStatus(ctx, input.ViewerID, input.Message)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	result := &apptypes.MessageResult{
		ID:                     input.Message.ID,
		RoomID:                 input.Message.RoomID,
		SenderID:               input.Message.SenderID,
		Message:                input.Message.Message,
		MessageType:            input.Message.MessageType,
		Status:                 status,
		MentionAll:             input.Message.MentionAll,
		Reactions:              buildMessageReactionResults(input.ViewerID, input.Message.Reactions),
		ReplyToMessageID:       input.Message.ReplyToMessageID,
		ForwardedFromMessageID: input.Message.ForwardedFromMessageID,
		FileName:               input.Message.FileName,
		FileSize:               input.Message.FileSize,
		MimeType:               input.Message.MimeType,
		ObjectKey:              input.Message.ObjectKey,
		DeletedForEveryone:     input.Message.DeletedForEveryoneAt != nil,
		CreatedAt:              input.Message.CreatedAt.UTC().Format(time.RFC3339),
	}
	if input.Message.EditedAt != nil {
		result.EditedAt = input.Message.EditedAt.UTC().Format(time.RFC3339)
	}
	if input.Message.DeletedForEveryoneAt != nil {
		result.Message = ""
	}

	if len(input.Message.Mentions) > 0 {
		result.Mentions = lo.Map(input.Message.Mentions, func(mention views.MessageMentionView, _ int) apptypes.MessageMentionResult {
			return apptypes.MessageMentionResult{
				AccountID:   mention.AccountID,
				DisplayName: mention.DisplayName,
				Username:    mention.Username,
			}
		})
	}

	if input.Message.ReplyToMessageID != "" {
		result.ReplyTo = b.buildMessagePreview(ctx, input.Message.ReplyToMessageID)
	}
	if input.Message.ForwardedFromMessageID != "" {
		result.ForwardedFrom = b.buildMessagePreview(ctx, input.Message.ForwardedFromMessageID)
	}

	return result, nil
}

func (b *messageQueryBuilder) resolveViewerStatus(ctx context.Context, viewerID string, message *views.MessageView) (string, error) {
	status := "sent"
	if strings.TrimSpace(message.SenderID) == strings.TrimSpace(viewerID) {
		seenCount, err := b.readRepos.MessageReadRepository().CountMessageReceiptsByStatus(ctx, message.ID, "seen")
		if err != nil {
			return "", stackErr.Error(err)
		}
		if seenCount > 0 {
			return "seen", nil
		}

		deliveredCount, err := b.readRepos.MessageReadRepository().CountMessageReceiptsByStatus(ctx, message.ID, "delivered")
		if err != nil {
			return "", stackErr.Error(err)
		}
		if deliveredCount > 0 {
			return "delivered", nil
		}
		return status, nil
	}

	receipt, err := b.readRepos.MessageReadRepository().GetMessageReceipt(ctx, projection.MessageReceiptLookup{
		MessageID: message.ID,
		AccountID: viewerID,
	})
	if err != nil {
		return "", stackErr.Error(err)
	}
	if receipt != nil && strings.TrimSpace(receipt.Status) != "" {
		return receipt.Status, nil
	}

	return status, nil
}

func (b *messageQueryBuilder) buildMessagePreview(ctx context.Context, messageID string) *apptypes.MessagePreviewResult {
	message, err := b.readRepos.MessageReadRepository().GetMessageByID(ctx, messageID)
	if err != nil || message == nil {
		return nil
	}

	return &apptypes.MessagePreviewResult{
		ID:          message.ID,
		SenderID:    message.SenderID,
		Message:     message.Message,
		MessageType: message.MessageType,
	}
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

func buildMessageReactionResults(viewerID string, items []views.MessageReactionView) []apptypes.MessageReactionResult {
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
	trimmedViewerID := strings.TrimSpace(viewerID)

	for _, item := range items {
		emoji := strings.TrimSpace(item.Emoji)
		accountID := strings.TrimSpace(item.AccountID)
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
		if accountID == trimmedViewerID {
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
