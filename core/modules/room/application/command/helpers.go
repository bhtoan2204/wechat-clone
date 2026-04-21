package command

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	roomsupport "wechat-clone/core/modules/room/application/support"
	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/modules/room/domain/aggregate"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/domain/repos"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

var mentionAllPattern = regexp.MustCompile(`(^|[[:space:][:punct:]])@all($|[[:space:][:punct:]])`)

type resolvedMessageMentions struct {
	Mentions            []entity.MessageMention
	OutboxMentions      []sharedevents.RoomMessageMention
	MentionAll          bool
	MentionedAccountIDs []string
}

func resolveMessageMentions(
	ctx context.Context,
	baseRepo repos.Repos,
	room *entity.Room,
	senderID string,
	command apptypes.SendMessageCommand,
	members []*entity.RoomMemberEntity,
) (*resolvedMessageMentions, error) {
	mentionAll := command.MentionAll || hasMentionAllToken(command.Message)
	explicitMentionIDs := normalizeMentionSelection(command.Mentions)

	if !room.IsGroup() {
		if mentionAll || len(explicitMentionIDs) > 0 {
			return nil, stackErr.Error(entity.ErrRoomMentionsRequireGroup)
		}
		return &resolvedMessageMentions{}, nil
	}

	memberSet := make(map[string]struct{}, len(members))
	mentionedAccountIDs := make([]string, 0, len(members))
	for _, member := range members {
		memberSet[member.AccountID] = struct{}{}
		if mentionAll && member.AccountID != senderID {
			mentionedAccountIDs = appendUniqueString(mentionedAccountIDs, member.AccountID)
		}
	}

	filteredExplicitIDs := make([]string, 0, len(explicitMentionIDs))
	for _, accountID := range explicitMentionIDs {
		if accountID == senderID {
			continue
		}
		if _, exists := memberSet[accountID]; !exists {
			return nil, stackErr.Error(entity.ErrRoomMentionTargetNotMember)
		}
		filteredExplicitIDs = append(filteredExplicitIDs, accountID)
		mentionedAccountIDs = appendUniqueString(mentionedAccountIDs, accountID)
	}

	if len(filteredExplicitIDs) == 0 {
		return &resolvedMessageMentions{
			MentionAll:          mentionAll,
			MentionedAccountIDs: mentionedAccountIDs,
		}, nil
	}

	accountProjections, err := baseRepo.RoomAccountRepository().ListByAccountIDs(ctx, filteredExplicitIDs)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountMap := lo.SliceToMap(accountProjections, func(projection *entity.AccountEntity) (string, *entity.AccountEntity) {
		if projection == nil {
			return "", nil
		}
		return projection.AccountID, projection
	})

	mentions := make([]entity.MessageMention, 0, len(filteredExplicitIDs))
	outboxMentions := make([]sharedevents.RoomMessageMention, 0, len(filteredExplicitIDs))
	for _, accountID := range filteredExplicitIDs {
		projection := accountMap[accountID]
		displayName := resolveMentionDisplayName(projection, accountID)
		username := ""
		if projection != nil {
			username = strings.TrimSpace(projection.Username)
		}

		mentions = append(mentions, entity.MessageMention{
			AccountID:   accountID,
			DisplayName: displayName,
			Username:    username,
		})
		outboxMentions = append(outboxMentions, sharedevents.RoomMessageMention{
			AccountID:   accountID,
			DisplayName: displayName,
			Username:    username,
		})
	}

	return &resolvedMessageMentions{
		Mentions:            mentions,
		OutboxMentions:      outboxMentions,
		MentionAll:          mentionAll,
		MentionedAccountIDs: mentionedAccountIDs,
	}, nil
}

func buildSenderIdentity(ctx context.Context, members []*entity.RoomMemberEntity, senderID string) aggregate.MessageSenderIdentity {
	actor, _ := actorctx.FromContext(ctx)

	identity := aggregate.MessageSenderIdentity{
		Name:  senderID,
		Email: "",
	}
	if actor != nil && actor.Email != "" {
		identity.Email = actor.Email
	}

	for _, member := range members {
		if member == nil || strings.TrimSpace(member.AccountID) != strings.TrimSpace(senderID) {
			continue
		}
		if displayName := resolveMentionDisplayName(&entity.AccountEntity{
			AccountID:   member.AccountID,
			DisplayName: member.DisplayName,
			Username:    member.Username,
		}, senderID); displayName != "" {
			identity.Name = displayName
		}
		return identity
	}

	if actor != nil && actor.Email != "" {
		identity.Name = actor.Email
	}

	return identity
}

func normalizeMentionSelection(mentions []apptypes.SendMessageMentionCommand) []string {
	if len(mentions) == 0 {
		return nil
	}

	normalized := lo.Uniq(lo.FilterMap(mentions, func(mention apptypes.SendMessageMentionCommand, _ int) (string, bool) {
		accountID := strings.TrimSpace(mention.AccountID)
		return accountID, accountID != ""
	}))
	return normalized
}

func hasMentionAllToken(message string) bool {
	return mentionAllPattern.MatchString(strings.ToLower(strings.TrimSpace(message)))
}

func resolveMentionDisplayName(account *entity.AccountEntity, fallback string) string {
	if account == nil {
		return strings.TrimSpace(fallback)
	}
	switch {
	case strings.TrimSpace(account.DisplayName) != "":
		return strings.TrimSpace(account.DisplayName)
	case strings.TrimSpace(account.Username) != "":
		return strings.TrimSpace(account.Username)
	default:
		return strings.TrimSpace(fallback)
	}
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

func lastPendingMessage(messages []*entity.MessageEntity) *entity.MessageEntity {
	if len(messages) == 0 {
		return nil
	}
	return messages[len(messages)-1]
}

func ensureProjectedAccountsExist(ctx context.Context, baseRepo repos.Repos, accountIDs ...string) error {
	normalizedIDs := lo.Uniq(lo.FilterMap(accountIDs, func(accountID string, _ int) (string, bool) {
		accountID = strings.TrimSpace(accountID)
		return accountID, accountID != ""
	}))
	if len(normalizedIDs) == 0 {
		return nil
	}

	accountProjections, err := baseRepo.RoomAccountRepository().ListByAccountIDs(ctx, normalizedIDs)
	if err != nil {
		return stackErr.Error(err)
	}

	projected := lo.SliceToMap(lo.Filter(accountProjections, func(projection *entity.AccountEntity, _ int) bool {
		return projection != nil
	}), func(projection *entity.AccountEntity) (string, struct{}) {
		return strings.TrimSpace(projection.AccountID), struct{}{}
	})

	for _, accountID := range normalizedIDs {
		if _, ok := projected[accountID]; ok {
			continue
		}
		return stackErr.Error(fmt.Errorf("%v: %s", ErrRoomAccountNotFound, accountID))
	}
	return nil
}

func executeSendMessage(ctx context.Context, baseRepo repos.Repos, accountID string, command apptypes.SendMessageCommand) (*apptypes.MessageResult, error) {
	roomAgg, err := baseRepo.RoomAggregateRepository().Load(ctx, strings.TrimSpace(command.RoomID))
	if err != nil {
		return nil, stackErr.Error(err)
	}

	mentions, err := resolveMessageMentions(ctx, baseRepo, roomAgg.Room(), accountID, command, roomAgg.Members())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	message, err := roomAgg.SendMessage(
		uuid.NewString(),
		accountID,
		entity.MessageParams{
			Message:                command.Message,
			MessageType:            command.MessageType,
			Mentions:               mentions.Mentions,
			MentionAll:             mentions.MentionAll,
			ReplyToMessageID:       command.ReplyToMessageID,
			ForwardedFromMessageID: command.ForwardedFromMessageID,
			FileName:               command.FileName,
			FileSize:               command.FileSize,
			MimeType:               command.MimeType,
			ObjectKey:              command.ObjectKey,
		},
		buildSenderIdentity(ctx, roomAgg.Members(), accountID),
		aggregate.MessageOutboxPayload{
			Mentions:            mentions.OutboxMentions,
			MentionAll:          mentions.MentionAll,
			MentionedAccountIDs: mentions.MentionedAccountIDs,
		},
		now,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, roomAgg))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return roomsupport.BuildMessageResultFromState(ctx, baseRepo, accountID, message)
}
