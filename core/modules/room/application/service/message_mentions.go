package service

import (
	"context"
	"regexp"
	"strings"

	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/stackErr"
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
	txRepos repos.Repos,
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

	accountProjections, err := txRepos.RoomAccountProjectionRepository().ListByAccountIDs(ctx, filteredExplicitIDs)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountMap := make(map[string]*entity.AccountEntity, len(accountProjections))
	for _, projection := range accountProjections {
		accountMap[projection.AccountID] = projection
	}

	mentions := make([]entity.MessageMention, 0, len(filteredExplicitIDs))
	outboxMentions := make([]sharedevents.RoomMessageMention, 0, len(filteredExplicitIDs))
	for _, accountID := range filteredExplicitIDs {
		projection := accountMap[accountID]
		displayName := resolveMentionDisplayName(projection, accountID)
		username := resolveMentionUsername(projection)

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

func buildSenderIdentity(ctx context.Context, txRepos repos.Repos, senderID string) (string, string) {
	actor, _ := currentActor(ctx)

	senderName := senderID
	senderEmail := ""
	if actor != nil && actor.Email != "" {
		senderName = actor.Email
		senderEmail = actor.Email
	}

	accountProjections, err := txRepos.RoomAccountProjectionRepository().ListByAccountIDs(ctx, []string{senderID})
	if err != nil || len(accountProjections) == 0 || accountProjections[0] == nil {
		return senderName, senderEmail
	}

	if displayName := resolveMentionDisplayName(accountProjections[0], senderID); displayName != "" {
		senderName = displayName
	}
	return senderName, senderEmail
}

func normalizeMentionSelection(mentions []apptypes.SendMessageMentionCommand) []string {
	if len(mentions) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(mentions))
	seen := make(map[string]struct{}, len(mentions))
	for _, mention := range mentions {
		accountID := strings.TrimSpace(mention.AccountID)
		if accountID == "" {
			continue
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		normalized = append(normalized, accountID)
	}
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

func resolveMentionUsername(account *entity.AccountEntity) string {
	if account == nil {
		return ""
	}
	return strings.TrimSpace(account.Username)
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
