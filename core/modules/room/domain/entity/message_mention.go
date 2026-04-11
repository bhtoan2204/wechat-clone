package entity

import (
	"errors"
	"strings"
)

var (
	ErrMessageMentionAccountRequired = errors.New("mention account_id is required")
)

type MessageMention struct {
	AccountID   string
	DisplayName string
	Username    string
}

func NormalizeMessageMentions(mentions []MessageMention) ([]MessageMention, error) {
	if len(mentions) == 0 {
		return nil, nil
	}

	normalized := make([]MessageMention, 0, len(mentions))
	seen := make(map[string]struct{}, len(mentions))
	for _, mention := range mentions {
		accountID := strings.TrimSpace(mention.AccountID)
		if accountID == "" {
			return nil, ErrMessageMentionAccountRequired
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		normalized = append(normalized, MessageMention{
			AccountID:   accountID,
			DisplayName: strings.TrimSpace(mention.DisplayName),
			Username:    strings.TrimSpace(mention.Username),
		})
	}

	return normalized, nil
}
