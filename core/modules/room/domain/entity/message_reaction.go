package entity

import (
	"errors"
	"sort"
	"strings"
	"time"

	"wechat-clone/core/shared/pkg/stackErr"
)

var (
	ErrMessageReactionEmojiRequired     = errors.New("emoji is required")
	ErrMessageReactionAccountIDRequired = errors.New("account_id is required")
)

type MessageReaction struct {
	AccountID string
	Emoji     string
	ReactedAt time.Time
}

func NormalizeMessageReactions(items []MessageReaction) ([]MessageReaction, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]MessageReaction, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		accountID := strings.TrimSpace(item.AccountID)
		emoji := strings.TrimSpace(item.Emoji)
		switch {
		case accountID == "":
			return nil, stackErr.Error(ErrMessageReactionAccountIDRequired)
		case emoji == "":
			return nil, stackErr.Error(ErrMessageReactionEmojiRequired)
		}

		reactedAt := item.ReactedAt.UTC()
		key := accountID + "\x00" + emoji
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		results = append(results, MessageReaction{
			AccountID: accountID,
			Emoji:     emoji,
			ReactedAt: reactedAt,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Emoji != results[j].Emoji {
			return results[i].Emoji < results[j].Emoji
		}
		if results[i].ReactedAt.Equal(results[j].ReactedAt) {
			return results[i].AccountID < results[j].AccountID
		}
		return results[i].ReactedAt.Before(results[j].ReactedAt)
	})

	return results, nil
}
